package testutil

import (
	"context"
	"time"

	"github.com/yantao-Wang/f1-guide/pkg/f1api"
)

// FakeStatsClient 实现 service.StatsClient，供 service / handler / contract 测试共用。
type FakeStatsClient struct {
	// Err 非空时所有方法直接返回该错误。
	Err error

	// 调用记录（含 season 实参），用于断言缺省赛季解析。
	ScheduleCalls             []int
	WinnersCalls              []int
	DriverStandingsCalls      []int
	ConstructorStandingsCalls []int
}

// SampleScheduleRaces 返回三场固定场景的比赛（相对 now：过去完赛、过去无赛果、未来）。
func SampleScheduleRaces(now time.Time) []f1api.ScheduleRace {
	past := now.AddDate(0, 0, -7).Format("2006-01-02")
	future := now.AddDate(0, 0, 7).Format("2006-01-02")
	return []f1api.ScheduleRace{
		{
			Round: "1", RaceName: "Australian Grand Prix", Date: past,
			Circuit: f1api.Circuit{CircuitName: "Albert Park Grand Prix Circuit", Location: f1api.Location{Country: "Australia"}},
		},
		{
			Round: "2", RaceName: "Chinese Grand Prix", Date: past,
			Circuit: f1api.Circuit{CircuitName: "Shanghai International Circuit", Location: f1api.Location{Country: "China"}},
		},
		{
			Round: "3", RaceName: "Japanese Grand Prix", Date: future,
			Circuit: f1api.Circuit{CircuitName: "Suzuka Circuit", Location: f1api.Location{Country: "Japan"}},
		},
	}
}

// Schedule 实现 StatsClient：赛程 + 冠军记录。
func (f *FakeStatsClient) Schedule(_ context.Context, season int) ([]f1api.ScheduleRace, error) {
	f.ScheduleCalls = append(f.ScheduleCalls, season)
	if f.Err != nil {
		return nil, f.Err
	}
	return SampleScheduleRaces(time.Now()), nil
}

// Winners 实现 StatsClient：仅第 1 站有冠军（第 2 站日期已过 → cancelled）。
func (f *FakeStatsClient) Winners(_ context.Context, season int) ([]f1api.WinnerRace, error) {
	f.WinnersCalls = append(f.WinnersCalls, season)
	if f.Err != nil {
		return nil, f.Err
	}
	return []f1api.WinnerRace{
		{
			Round: "1",
			Results: []f1api.WinnerEntry{
				{Position: "1", Driver: f1api.Driver{DriverID: "max_verstappen", GivenName: "Max", FamilyName: "Verstappen"}},
			},
		},
	}, nil
}

// DriverStandings 实现 StatsClient：六行（含未知车手回退场景）。
func (f *FakeStatsClient) DriverStandings(_ context.Context, season int) ([]f1api.Standing, error) {
	f.DriverStandingsCalls = append(f.DriverStandingsCalls, season)
	if f.Err != nil {
		return nil, f.Err
	}
	return []f1api.Standing{
		{
			Position: "1", Points: "267", Wins: "7",
			Driver:       f1api.Driver{DriverID: "max_verstappen", GivenName: "Max", FamilyName: "Verstappen"},
			Constructors: []f1api.Constructor{{Name: "Red Bull"}},
		},
		{
			Position: "2", Points: "201", Wins: "2",
			Driver:       f1api.Driver{DriverID: "russell", GivenName: "George", FamilyName: "Russell"},
			Constructors: []f1api.Constructor{{Name: "Mercedes"}},
		},
		{
			Position: "3", Points: "191", Wins: "1",
			Driver:       f1api.Driver{DriverID: "hamilton", GivenName: "Lewis", FamilyName: "Hamilton"},
			Constructors: []f1api.Constructor{{Name: "Ferrari"}},
		},
		{
			Position: "4", Points: "171", Wins: "2",
			Driver:       f1api.Driver{DriverID: "norris", GivenName: "Lando", FamilyName: "Norris"},
			Constructors: []f1api.Constructor{{Name: "McLaren"}},
		},
		{
			Position: "5", Points: "155", Wins: "1",
			Driver:       f1api.Driver{DriverID: "leclerc", GivenName: "Charles", FamilyName: "Leclerc"},
			Constructors: []f1api.Constructor{{Name: "Ferrari"}},
		},
		{
			Position: "6", Points: "10", Wins: "0",
			Driver:       f1api.Driver{DriverID: "unknown_driver", GivenName: "Nemo", FamilyName: "Unknown"},
			Constructors: []f1api.Constructor{{Name: "Team X"}},
		},
	}, nil
}

// ConstructorStandings 实现 StatsClient。
func (f *FakeStatsClient) ConstructorStandings(_ context.Context, season int) ([]f1api.ConstructorStanding, error) {
	f.ConstructorStandingsCalls = append(f.ConstructorStandingsCalls, season)
	if f.Err != nil {
		return nil, f.Err
	}
	return []f1api.ConstructorStanding{
		{Position: "1", Points: "468", Wins: "9", Constructor: f1api.Constructor{Name: "Mercedes"}},
		{Position: "2", Points: "346", Wins: "2", Constructor: f1api.Constructor{Name: "Ferrari"}},
		{Position: "11", Points: "0", Wins: "0", Constructor: f1api.Constructor{Name: "Cadillac F1 Team"}},
	}, nil
}
