// 后台内容管理：车手 / 赛道 / 名场面三表 CRUD 处理器。
//
// 全部遵循 PRG 模式：成功 303 到列表页（携带 flash 参数），
// 校验失败 422 / 冲突 409 重渲染表单并保留输入。
package handler

import (
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/yantao-Wang/f1-guide/internal/domain"
	"github.com/yantao-Wang/f1-guide/internal/service"
)

// parseAdminForm 解析后台表单：multipart（含文件上传）走 ParseMultipartForm（内存上限 8MB），
// 其余走 ParseForm。超限或格式错误返回 false 并已写出 413 响应。
func (h *Admin) parseAdminForm(w http.ResponseWriter, r *http.Request) bool {
	var err error
	if strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/") {
		err = r.ParseMultipartForm(maxImageBytes)
	} else {
		err = r.ParseForm()
	}
	if err != nil {
		h.log.Error("admin parse form", "error", err, "content_type", r.Header.Get("Content-Type"))
		http.Error(w, "表单数据过大或格式错误", http.StatusRequestEntityTooLarge)
		return false
	}
	return true
}

// --- 车手 ---

// driverListBody 车手管理列表数据。
type driverListBody struct {
	Drivers []domain.DriverSummary
}

// driverFormBody 车手表单页数据（新增/编辑共用模板）。
type driverFormBody struct {
	Form         driverForm
	IsEdit       bool
	OldSlug      string
	CurrentImage string
}

// DriverList 车手管理列表。
func (h *Admin) DriverList(w http.ResponseWriter, r *http.Request) {
	items, err := h.svc.ListDrivers(r.Context(), nil)
	if err != nil {
		renderPageError(w, err)
		return
	}
	h.renderAdmin(w, r, "admin_drivers", "车手管理", driverListBody{Drivers: items}, adminFlash(r), "")
}

// DriverNew 新增车手表单。
func (h *Admin) DriverNew(w http.ResponseWriter, r *http.Request) {
	h.renderAdmin(w, r, "admin_driver_form", "新增车手", driverFormBody{Form: driverForm{Errors: map[string]string{}}}, "", "")
}

// DriverCreate 提交新增车手。
func (h *Admin) DriverCreate(w http.ResponseWriter, r *http.Request) {
	if !h.parseAdminForm(w, r) {
		return
	}
	form := newDriverForm(r)
	imageURL, uploadErr := h.saveUpload(r, "image", "")
	if uploadErr != nil {
		form.Errors["image"] = uploadErr.Error()
	}
	form.validate()
	if len(form.Errors) > 0 {
		h.renderAdminStatus(w, r, http.StatusUnprocessableEntity, "admin_driver_form", "新增车手",
			driverFormBody{Form: form}, "", "表单校验未通过，请修正后重试")
		return
	}

	d := form.toDriver()
	d.ImageURL = imageURL
	if err := h.admin.CreateDriver(r.Context(), d); err != nil {
		if errors.Is(err, service.ErrConflict) {
			form.Errors["slug"] = "slug 或 Jolpica ID 已被占用"
			h.renderAdminStatus(w, r, http.StatusConflict, "admin_driver_form", "新增车手",
				driverFormBody{Form: form}, "", "slug 或 Jolpica ID 已被占用")
			return
		}
		h.log.Error("admin create driver", "error", err)
		renderPageError(w, err)
		return
	}
	h.log.Info("admin create driver", "slug", d.Slug, "ip", r.RemoteAddr)
	adminRedirect(w, r, "drivers", "created", d.Slug)
}

// DriverEdit 编辑车手表单（回填现有数据）。
func (h *Admin) DriverEdit(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	d, err := h.svc.GetDriver(r.Context(), slug)
	if errors.Is(err, service.ErrNotFound) {
		renderNotFound(w)
		return
	}
	if err != nil {
		renderPageError(w, err)
		return
	}
	h.renderAdmin(w, r, "admin_driver_form", "编辑车手",
		driverFormBody{Form: driverFormFrom(d), IsEdit: true, OldSlug: slug, CurrentImage: d.ImageURL}, "", "")
}

// DriverUpdate 提交编辑车手。
func (h *Admin) DriverUpdate(w http.ResponseWriter, r *http.Request) {
	if !h.parseAdminForm(w, r) {
		return
	}
	oldSlug := chi.URLParam(r, "slug")
	existing, err := h.svc.GetDriver(r.Context(), oldSlug)
	if errors.Is(err, service.ErrNotFound) {
		renderNotFound(w)
		return
	}
	if err != nil {
		renderPageError(w, err)
		return
	}

	form := newDriverForm(r)
	imageURL, uploadErr := h.saveUpload(r, "image", existing.ImageURL)
	if uploadErr != nil {
		form.Errors["image"] = uploadErr.Error()
	}
	form.validate()
	if len(form.Errors) > 0 {
		h.renderAdminStatus(w, r, http.StatusUnprocessableEntity, "admin_driver_form", "编辑车手",
			driverFormBody{Form: form, IsEdit: true, OldSlug: oldSlug, CurrentImage: existing.ImageURL}, "", "表单校验未通过，请修正后重试")
		return
	}

	d := form.toDriver()
	if form.RemoveImage {
		d.ImageURL = ""
	} else {
		d.ImageURL = imageURL
	}
	if err := h.admin.UpdateDriver(r.Context(), oldSlug, d); err != nil {
		if errors.Is(err, service.ErrNotFound) {
			renderNotFound(w)
			return
		}
		if errors.Is(err, service.ErrConflict) {
			form.Errors["slug"] = "slug 或 Jolpica ID 已被占用"
			h.renderAdminStatus(w, r, http.StatusConflict, "admin_driver_form", "编辑车手",
				driverFormBody{Form: form, IsEdit: true, OldSlug: oldSlug, CurrentImage: existing.ImageURL}, "", "slug 或 Jolpica ID 已被占用")
			return
		}
		h.log.Error("admin update driver", "slug", oldSlug, "error", err)
		renderPageError(w, err)
		return
	}
	// 照片被更换或清除：删除旧文件（尽力而为）
	if d.ImageURL != existing.ImageURL && existing.ImageURL != "" {
		h.removeUploadFile(existing.ImageURL)
	}
	h.log.Info("admin update driver", "slug", d.Slug, "ip", r.RemoteAddr)
	adminRedirect(w, r, "drivers", "saved", d.Slug)
}

// DriverDelete 删除车手。
func (h *Admin) DriverDelete(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	existing, err := h.svc.GetDriver(r.Context(), slug)
	if errors.Is(err, service.ErrNotFound) {
		renderNotFound(w)
		return
	}
	if err != nil {
		renderPageError(w, err)
		return
	}
	if err := h.admin.DeleteDriver(r.Context(), slug); err != nil {
		if errors.Is(err, service.ErrNotFound) {
			renderNotFound(w)
			return
		}
		h.log.Error("admin delete driver", "slug", slug, "error", err)
		renderPageError(w, err)
		return
	}
	if existing.ImageURL != "" {
		h.removeUploadFile(existing.ImageURL)
	}
	h.log.Info("admin delete driver", "slug", slug, "ip", r.RemoteAddr)
	adminRedirect(w, r, "drivers", "deleted", slug)
}

// --- 赛道 ---

// trackListBody 赛道管理列表数据。
type trackListBody struct {
	Tracks []domain.TrackSummary
}

// trackFormBody 赛道表单页数据。
type trackFormBody struct {
	Form         trackForm
	IsEdit       bool
	OldSlug      string
	CurrentImage string
}

// TrackList 赛道管理列表。
func (h *Admin) TrackList(w http.ResponseWriter, r *http.Request) {
	items, err := h.svc.ListTracks(r.Context())
	if err != nil {
		renderPageError(w, err)
		return
	}
	h.renderAdmin(w, r, "admin_tracks", "赛道管理", trackListBody{Tracks: items}, adminFlash(r), "")
}

// TrackNew 新增赛道表单。
func (h *Admin) TrackNew(w http.ResponseWriter, r *http.Request) {
	h.renderAdmin(w, r, "admin_track_form", "新增赛道", trackFormBody{Form: trackForm{Errors: map[string]string{}}}, "", "")
}

// TrackCreate 提交新增赛道。
func (h *Admin) TrackCreate(w http.ResponseWriter, r *http.Request) {
	if !h.parseAdminForm(w, r) {
		return
	}
	form := newTrackForm(r)
	imageURL, uploadErr := h.saveUpload(r, "image", "")
	if uploadErr != nil {
		form.Errors["image"] = uploadErr.Error()
	}
	form.validate()
	if len(form.Errors) > 0 {
		h.renderAdminStatus(w, r, http.StatusUnprocessableEntity, "admin_track_form", "新增赛道",
			trackFormBody{Form: form}, "", "表单校验未通过，请修正后重试")
		return
	}

	t := form.toTrack()
	t.CircuitMapURL = imageURL
	if err := h.admin.CreateTrack(r.Context(), t); err != nil {
		if errors.Is(err, service.ErrConflict) {
			form.Errors["slug"] = "slug 已被占用"
			h.renderAdminStatus(w, r, http.StatusConflict, "admin_track_form", "新增赛道",
				trackFormBody{Form: form}, "", "slug 已被占用")
			return
		}
		h.log.Error("admin create track", "error", err)
		renderPageError(w, err)
		return
	}
	h.log.Info("admin create track", "slug", t.Slug, "ip", r.RemoteAddr)
	adminRedirect(w, r, "tracks", "created", t.Slug)
}

// TrackEdit 编辑赛道表单。
func (h *Admin) TrackEdit(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	t, err := h.svc.GetTrack(r.Context(), slug)
	if errors.Is(err, service.ErrNotFound) {
		renderNotFound(w)
		return
	}
	if err != nil {
		renderPageError(w, err)
		return
	}
	h.renderAdmin(w, r, "admin_track_form", "编辑赛道",
		trackFormBody{Form: trackFormFrom(t), IsEdit: true, OldSlug: slug, CurrentImage: t.CircuitMapURL}, "", "")
}

// TrackUpdate 提交编辑赛道。
func (h *Admin) TrackUpdate(w http.ResponseWriter, r *http.Request) {
	if !h.parseAdminForm(w, r) {
		return
	}
	oldSlug := chi.URLParam(r, "slug")
	existing, err := h.svc.GetTrack(r.Context(), oldSlug)
	if errors.Is(err, service.ErrNotFound) {
		renderNotFound(w)
		return
	}
	if err != nil {
		renderPageError(w, err)
		return
	}

	form := newTrackForm(r)
	imageURL, uploadErr := h.saveUpload(r, "image", existing.CircuitMapURL)
	if uploadErr != nil {
		form.Errors["image"] = uploadErr.Error()
	}
	form.validate()
	if len(form.Errors) > 0 {
		h.renderAdminStatus(w, r, http.StatusUnprocessableEntity, "admin_track_form", "编辑赛道",
			trackFormBody{Form: form, IsEdit: true, OldSlug: oldSlug, CurrentImage: existing.CircuitMapURL}, "", "表单校验未通过，请修正后重试")
		return
	}

	t := form.toTrack()
	if form.RemoveImage {
		t.CircuitMapURL = ""
	} else {
		t.CircuitMapURL = imageURL
	}
	if err := h.admin.UpdateTrack(r.Context(), oldSlug, t); err != nil {
		if errors.Is(err, service.ErrNotFound) {
			renderNotFound(w)
			return
		}
		if errors.Is(err, service.ErrConflict) {
			form.Errors["slug"] = "slug 已被占用"
			h.renderAdminStatus(w, r, http.StatusConflict, "admin_track_form", "编辑赛道",
				trackFormBody{Form: form, IsEdit: true, OldSlug: oldSlug, CurrentImage: existing.CircuitMapURL}, "", "slug 已被占用")
			return
		}
		h.log.Error("admin update track", "slug", oldSlug, "error", err)
		renderPageError(w, err)
		return
	}
	if t.CircuitMapURL != existing.CircuitMapURL && existing.CircuitMapURL != "" {
		h.removeUploadFile(existing.CircuitMapURL)
	}
	h.log.Info("admin update track", "slug", t.Slug, "ip", r.RemoteAddr)
	adminRedirect(w, r, "tracks", "saved", t.Slug)
}

// TrackDelete 删除赛道（关联名场面自动解绑）。
func (h *Admin) TrackDelete(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	existing, err := h.svc.GetTrack(r.Context(), slug)
	if errors.Is(err, service.ErrNotFound) {
		renderNotFound(w)
		return
	}
	if err != nil {
		renderPageError(w, err)
		return
	}
	if err := h.admin.DeleteTrack(r.Context(), slug); err != nil {
		if errors.Is(err, service.ErrNotFound) {
			renderNotFound(w)
			return
		}
		h.log.Error("admin delete track", "slug", slug, "error", err)
		renderPageError(w, err)
		return
	}
	if existing.CircuitMapURL != "" {
		h.removeUploadFile(existing.CircuitMapURL)
	}
	h.log.Info("admin delete track", "slug", slug, "ip", r.RemoteAddr)
	adminRedirect(w, r, "tracks", "deleted", slug)
}

// --- 名场面 ---

// momentListBody 名场面管理列表数据。
type momentListBody struct {
	Moments []momentCard
}

// momentFormBody 名场面表单页数据（含关联赛道/车手选项）。
type momentFormBody struct {
	Form    momentForm
	IsEdit  bool
	OldSlug string
	Tracks  []domain.TrackSummary
	Drivers []domain.DriverSummary
}

// momentOptions 加载名场面表单的关联选项（赛道下拉 + 车手多选）。
func (h *Admin) momentOptions(r *http.Request) ([]domain.TrackSummary, []domain.DriverSummary, error) {
	tracks, err := h.svc.ListTracks(r.Context())
	if err != nil {
		return nil, nil, err
	}
	drivers, err := h.svc.ListDrivers(r.Context(), nil)
	if err != nil {
		return nil, nil, err
	}
	return tracks, drivers, nil
}

// MomentList 名场面管理列表。
func (h *Admin) MomentList(w http.ResponseWriter, r *http.Request) {
	items, err := h.svc.ListMoments(r.Context())
	if err != nil {
		renderPageError(w, err)
		return
	}
	cards := make([]momentCard, 0, len(items))
	for _, m := range items {
		cards = append(cards, momentCard{MomentSummary: m, TypeLabel: momentTypeLabel(m.Type)})
	}
	h.renderAdmin(w, r, "admin_moments", "名场面管理", momentListBody{Moments: cards}, adminFlash(r), "")
}

// MomentNew 新增名场面表单。
func (h *Admin) MomentNew(w http.ResponseWriter, r *http.Request) {
	tracks, drivers, err := h.momentOptions(r)
	if err != nil {
		renderPageError(w, err)
		return
	}
	h.renderAdmin(w, r, "admin_moment_form", "新增名场面",
		momentFormBody{Form: momentForm{Errors: map[string]string{}}, Tracks: tracks, Drivers: drivers}, "", "")
}

// MomentCreate 提交新增名场面。
func (h *Admin) MomentCreate(w http.ResponseWriter, r *http.Request) {
	if !h.parseAdminForm(w, r) {
		return
	}
	form := newMomentForm(r)
	form.validate()
	if len(form.Errors) > 0 {
		tracks, drivers, err := h.momentOptions(r)
		if err != nil {
			renderPageError(w, err)
			return
		}
		h.renderAdminStatus(w, r, http.StatusUnprocessableEntity, "admin_moment_form", "新增名场面",
			momentFormBody{Form: form, Tracks: tracks, Drivers: drivers}, "", "表单校验未通过，请修正后重试")
		return
	}

	m := form.toMoment()
	if err := h.admin.CreateMoment(r.Context(), m); err != nil {
		if errors.Is(err, service.ErrConflict) {
			form.Errors["slug"] = "slug 已被占用"
			tracks, drivers, optErr := h.momentOptions(r)
			if optErr != nil {
				renderPageError(w, optErr)
				return
			}
			h.renderAdminStatus(w, r, http.StatusConflict, "admin_moment_form", "新增名场面",
				momentFormBody{Form: form, Tracks: tracks, Drivers: drivers}, "", "slug 已被占用")
			return
		}
		h.log.Error("admin create moment", "error", err)
		renderPageError(w, err)
		return
	}
	h.log.Info("admin create moment", "slug", m.Slug, "ip", r.RemoteAddr)
	adminRedirect(w, r, "moments", "created", m.Slug)
}

// MomentEdit 编辑名场面表单。
func (h *Admin) MomentEdit(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	m, err := h.svc.GetMoment(r.Context(), slug)
	if errors.Is(err, service.ErrNotFound) {
		renderNotFound(w)
		return
	}
	if err != nil {
		renderPageError(w, err)
		return
	}
	tracks, drivers, err := h.momentOptions(r)
	if err != nil {
		renderPageError(w, err)
		return
	}
	h.renderAdmin(w, r, "admin_moment_form", "编辑名场面",
		momentFormBody{Form: momentFormFrom(m), IsEdit: true, OldSlug: slug, Tracks: tracks, Drivers: drivers}, "", "")
}

// MomentUpdate 提交编辑名场面。
func (h *Admin) MomentUpdate(w http.ResponseWriter, r *http.Request) {
	if !h.parseAdminForm(w, r) {
		return
	}
	oldSlug := chi.URLParam(r, "slug")
	form := newMomentForm(r)
	form.validate()
	if len(form.Errors) > 0 {
		tracks, drivers, err := h.momentOptions(r)
		if err != nil {
			renderPageError(w, err)
			return
		}
		h.renderAdminStatus(w, r, http.StatusUnprocessableEntity, "admin_moment_form", "编辑名场面",
			momentFormBody{Form: form, IsEdit: true, OldSlug: oldSlug, Tracks: tracks, Drivers: drivers}, "", "表单校验未通过，请修正后重试")
		return
	}

	m := form.toMoment()
	if err := h.admin.UpdateMoment(r.Context(), oldSlug, m); err != nil {
		if errors.Is(err, service.ErrNotFound) {
			renderNotFound(w)
			return
		}
		if errors.Is(err, service.ErrConflict) {
			form.Errors["slug"] = "slug 已被占用"
			tracks, drivers, optErr := h.momentOptions(r)
			if optErr != nil {
				renderPageError(w, optErr)
				return
			}
			h.renderAdminStatus(w, r, http.StatusConflict, "admin_moment_form", "编辑名场面",
				momentFormBody{Form: form, IsEdit: true, OldSlug: oldSlug, Tracks: tracks, Drivers: drivers}, "", "slug 已被占用")
			return
		}
		h.log.Error("admin update moment", "slug", oldSlug, "error", err)
		renderPageError(w, err)
		return
	}
	h.log.Info("admin update moment", "slug", m.Slug, "ip", r.RemoteAddr)
	adminRedirect(w, r, "moments", "saved", m.Slug)
}

// MomentDelete 删除名场面。
func (h *Admin) MomentDelete(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	if err := h.admin.DeleteMoment(r.Context(), slug); err != nil {
		if errors.Is(err, service.ErrNotFound) {
			renderNotFound(w)
			return
		}
		h.log.Error("admin delete moment", "slug", slug, "error", err)
		renderPageError(w, err)
		return
	}
	h.log.Info("admin delete moment", "slug", slug, "ip", r.RemoteAddr)
	adminRedirect(w, r, "moments", "deleted", slug)
}
