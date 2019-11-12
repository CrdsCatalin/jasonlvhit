package gocron

import (
	"fmt"
	"log"
<<<<<<< HEAD
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
=======
	"sync/atomic"
	"testing"
	"time"

	"github.com/alicebob/miniredis"
	"github.com/go-redis/redis"
>>>>>>> prod-safety
)

var defaultOption = func(j *Job) {
	j.ShouldDoImmediately = true
}

func task() {
	fmt.Println("I am a running job.")
}

func taskWithParams(a int, b string) {
	fmt.Println(a, b)
}

<<<<<<< HEAD
func TestSecond(t *testing.T) {
	sched := NewScheduler()
	job := sched.Every(1).Second()
	testJobWithInterval(t, sched, job, 1)
}

func TestSeconds(t *testing.T) {
	sched := NewScheduler()
	job := sched.Every(2).Seconds()
	testJobWithInterval(t, sched, job, 2)
}

func testJobWithInterval(t *testing.T, sched *Scheduler, job *Job, expectedTimeBetweenRuns int64) {
	jobDone := make(chan bool)
	executionTimes := make([]int64, 0)
	numberOfIterations := 2

	job.Do(func() {
		executionTimes = append(executionTimes, time.Now().Unix())
		if len(executionTimes) >= numberOfIterations {
			jobDone <- true
		}
	})

	stop := sched.Start()
	<-jobDone // Wait job done
	close(stop)

	assert.Equal(t, numberOfIterations, len(executionTimes), "did not run expected number of times")

	for i := 1; i < numberOfIterations; i++ {
		durationBetweenExecutions := executionTimes[i] - executionTimes[i-1]
		assert.Equal(t, expectedTimeBetweenRuns, durationBetweenExecutions, "Duration between tasks does not correspond to expectations")
	}
}

func TestSafeExecution(t *testing.T) {
	sched := NewScheduler()
	success := false
	sched.Every(1).Second().Do(func(mutableValue *bool) {
		*mutableValue = !*mutableValue
	}, &success)
	sched.RunAll()
	assert.Equal(t, true, success, "Task did not get called")
}

func TestSafeExecutionWithPanic(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Unexpected internal panic occurred: %s", r)
		}
	}()

	sched := NewScheduler()
	sched.Every(1).Second().DoSafely(func() {
		log.Panic("I am panicking!")
	})
	sched.RunAll()
}

func TestScheduled(t *testing.T) {
	n := NewScheduler()
	n.Every(1).Second().Do(task)
	if !n.Scheduled(task) {
		t.Fatal("Task was scheduled but function couldn't find it")
	}
}

// This is a basic test for the issue described here: https://github.com/jasonlvhit/gocron/issues/23
func TestScheduler_Weekdays(t *testing.T) {
	scheduler := NewScheduler()

	job1 := scheduler.Every(1).Monday().At("23:59")
	job2 := scheduler.Every(1).Wednesday().At("23:59")
	job1.Do(task)
	job2.Do(task)
	t.Logf("job1 scheduled for %s", job1.NextScheduledTime())
	t.Logf("job2 scheduled for %s", job2.NextScheduledTime())
	assert.NotEqual(t, job1.NextScheduledTime(), job2.NextScheduledTime(), "Two jobs scheduled at the same time on two different weekdays should never run at the same time")
}

// This ensures that if you schedule a job for today's weekday, but the time is already passed, it will be scheduled for
// next week at the requested time.
func TestScheduler_WeekdaysTodayAfter(t *testing.T) {
	scheduler := NewScheduler()

	now := time.Now()
	timeToSchedule := time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), now.Minute()-1, 0, 0, time.Local)

	job := callTodaysWeekday(scheduler.Every(1)).At(fmt.Sprintf("%02d:%02d", timeToSchedule.Hour(), timeToSchedule.Minute()))
	job.Do(task)
	t.Logf("job is scheduled for %s", job.NextScheduledTime())
	if job.NextScheduledTime().Weekday() != timeToSchedule.Weekday() {
		t.Errorf("Job scheduled for current weekday for earlier time, should still be scheduled for current weekday (but next week)")
	}
	nextWeek := time.Date(now.Year(), now.Month(), now.Day()+7, now.Hour(), now.Minute()-1, 0, 0, time.Local)
	if !job.NextScheduledTime().Equal(nextWeek) {
		t.Errorf("Job should be scheduled for the correct time next week.\nGot %+v, expected %+v", job.NextScheduledTime(), nextWeek)
	}
}

func TestScheduler_JobLocsSetProperly(t *testing.T) {
	defaultScheduledJob := NewJob(10)
	assert.Equal(t, defaultScheduledJob.loc, time.Local)
	defaultScheduledJobFromScheduler := Every(10)
	assert.Equal(t, defaultScheduledJobFromScheduler.loc, time.Local)

	laLocation, err := time.LoadLocation("America/Los_Angeles")
	if err != nil {
		t.Fatalf("unable to load America/Los_Angeles time location")
	}

	ChangeLoc(laLocation)
	modifiedGlobalLocJob := NewJob(10)
	assert.Equal(t, modifiedGlobalLocJob.loc, laLocation)
	modifiedGlobalLocJobFromScheduler := Every(10)
	assert.Equal(t, modifiedGlobalLocJobFromScheduler.loc, laLocation)

	chiLocation, err := time.LoadLocation("America/Chicago")
	if err != nil {
		t.Fatalf("unable to load America/Chicago time location")
	}

	scheduler := NewScheduler()
	scheduler.ChangeLoc(chiLocation)
	modifiedGlobalLocJobFromScheduler = scheduler.Every(10)
	assert.Equal(t, modifiedGlobalLocJobFromScheduler.loc, chiLocation)

	// other tests depend on global state :(
	ChangeLoc(time.Local)
}

func TestScheduleNextRunLoc(t *testing.T) {
	laLocation, err := time.LoadLocation("America/Los_Angeles")
	if err != nil {
		t.Fatalf("unable to load America/Los_Angeles time location")
	}

	sched := NewScheduler()
	sched.ChangeLoc(time.UTC)

	job := sched.Every(1).Day().At("20:44")

	// job just ran (this is 20:45 UTC), so next run should be tomorrow
	today := time.Now().In(laLocation)
	job.lastRun = time.Date(today.Year(), today.Month(), today.Day(), 13, 45, 0, 0, laLocation)
	job.Do(task)

	tomorrow := today.AddDate(0, 0, 1)
	assert.Equal(t, 20, job.NextScheduledTime().UTC().Hour())
	assert.Equal(t, 44, job.NextScheduledTime().UTC().Minute())
	assert.Equal(t, tomorrow.Day(), job.NextScheduledTime().UTC().Day())
}

func TestScheduleNextRunFromNow(t *testing.T) {
	now := time.Now()

	sched := NewScheduler()
	sched.ChangeLoc(time.UTC)

	job := sched.Every(1).Hour().From(NextTick())
	job.Do(task)

	next := job.NextScheduledTime()
	nextRounded := time.Date(next.Year(), next.Month(), next.Day(), next.Hour(), next.Minute(), next.Second(), 0, time.UTC)

	expected := time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), now.Minute(), now.Second(), 0, time.UTC).Add(time.Second)

	assert.Exactly(t, expected, nextRounded)
}

// This is to ensure that if you schedule a job for today's weekday, and the time hasn't yet passed, the next run time
// will be scheduled for today.
func TestScheduler_WeekdaysTodayBefore(t *testing.T) {
	scheduler := NewScheduler()

	now := time.Now()
	timeToSchedule := time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), now.Minute()+1, 0, 0, time.Local)

	job := callTodaysWeekday(scheduler.Every(1)).At(fmt.Sprintf("%02d:%02d", timeToSchedule.Hour(), timeToSchedule.Minute()))
	job.Do(task)
	t.Logf("job is scheduled for %s", job.NextScheduledTime())
	if !job.NextScheduledTime().Equal(timeToSchedule) {
		t.Error("Job should be run today, at the set time.")
	}
=======
func assertEqualTime(name string, t *testing.T, actual, expected time.Time) {
	if actual != expected {
		t.Errorf("test name: %s actual different than expected want: %v -> got: %v", name, expected, actual)
	}
}

func TestSecond(t *testing.T) {

	defaultScheduler.Every(1, defaultOption).Second().Do(task)
	defaultScheduler.Every(1, defaultOption).Second().Do(taskWithParams, 1, "hello")
	stop := defaultScheduler.Start()
	time.Sleep(5 * time.Second)
	close(stop)

	if err := defaultScheduler.Err(); err != nil {
		t.Error(err)
	}
	defaultScheduler.Clear()
>>>>>>> prod-safety
}

func Test_formatTime(t *testing.T) {
	tests := []struct {
		name     string
		args     string
		wantHour int
		wantMin  int
		wantErr  bool
	}{
		{
			name:     "normal",
			args:     "16:18",
			wantHour: 16,
			wantMin:  18,
			wantErr:  false,
		},
		{
			name:     "normal",
			args:     "6:18",
			wantHour: 6,
			wantMin:  18,
			wantErr:  false,
		},
		{
			name:     "not_a_number",
			args:     "e:18",
			wantHour: 0,
			wantMin:  0,
			wantErr:  true,
		},
		{
			name:     "out_of_range_hour",
			args:     "25:18",
			wantHour: 0,
			wantMin:  0,
			wantErr:  true,
		},
		{
			name:     "out_of_range_minute",
			args:     "23:60",
			wantHour: 0,
			wantMin:  0,
			wantErr:  true,
		},
		{
			name:     "wrong_format",
			args:     "19:18:17",
			wantHour: 0,
			wantMin:  0,
			wantErr:  true,
		},
		{
			name:     "wrong_minute",
			args:     "19:1e",
			wantHour: 19,
			wantMin:  0,
			wantErr:  true,
		},
		{
			name:     "wrong_hour",
			args:     "1e:10",
			wantHour: 11,
			wantMin:  0,
			wantErr:  true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotHour, gotMin, err := formatTime(tt.args)
			if tt.wantErr {
				assert.NotEqual(t, nil, err, tt.args)
				return
			}
			assert.Equal(t, tt.wantHour, gotHour, tt.args)
			assert.Equal(t, tt.wantMin, gotMin, tt.args)
		})
	}
}

// utility function for testing the weekday functions *on* the current weekday.
func callTodaysWeekday(job *Job) *Job {
	switch time.Now().Weekday() {
	case 0:
		job.Sunday()
	case 1:
		job.Monday()
	case 2:
		job.Tuesday()
	case 3:
		job.Wednesday()
	case 4:
		job.Thursday()
	case 5:
		job.Friday()
	case 6:
		job.Saturday()
	}
	return job
}

func TestScheduler_Remove(t *testing.T) {
	scheduler := NewScheduler()
	scheduler.Every(1).Minute().Do(task)
	scheduler.Every(1).Minute().Do(taskWithParams, 1, "hello")

	assert.Equal(t, 2, scheduler.Len(), "Incorrect number of jobs")

	scheduler.Remove(task)

	assert.Equal(t, 1, scheduler.Len(), "Incorrect number of jobs after removing 1 job")

	scheduler.Remove(task)

	assert.Equal(t, 1, scheduler.Len(), "Incorrect number of jobs after removing non-existent job")
}

func TestScheduler_RemoveByRef(t *testing.T) {
	scheduler := NewScheduler()
	job1 := scheduler.Every(1).Minute()
	job1.Do(task)
	job2 := scheduler.Every(1).Minute()
	job2.Do(taskWithParams, 1, "hello")

	assert.Equal(t, 2, scheduler.Len(), "Incorrect number of jobs")

	scheduler.RemoveByRef(job1)
	assert.ElementsMatch(t, []*Job{job2}, scheduler.Jobs())
}

func TestTaskAt(t *testing.T) {
	// Create new scheduler to have clean test env
	s := NewScheduler()

	// Schedule to run in next minute
	now := time.Now()
	// Schedule every day At
<<<<<<< HEAD
	startAt := fmt.Sprintf("%02d:%02d", now.Hour(), now.Add(time.Minute).Minute())
	dayJob := s.Every(1).Day().At(startAt)
=======
	startAt := fmt.Sprintf("%02d:%02d", now.Hour(), now.Minute()+1)
	dayJob := s.Every(1, defaultOption).Day().At(startAt)
	if err := dayJob.Err(); err != nil {
		t.Error(err)
	}
>>>>>>> prod-safety

	dayJobDone := make(chan bool, 1)
	dayJob.Do(func() {
		dayJobDone <- true
	})

	// Expected start time
	expectedStartTime := time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), now.Add(time.Minute).Minute(), 0, 0, loc)
	nextRun := dayJob.NextScheduledTime()
<<<<<<< HEAD
	assert.Equal(t, expectedStartTime, nextRun)
=======
	assertEqualTime("first run", t, nextRun, startTime)
>>>>>>> prod-safety

	sStop := s.Start()
	<-dayJobDone // Wait job done
	close(sStop)
	time.Sleep(time.Second) // wait for scheduler to reschedule job

	// Expected next start time 1 day after
	expectedNextRun := expectedStartTime.AddDate(0, 0, 1)
	nextRun = dayJob.NextScheduledTime()
<<<<<<< HEAD
	assert.Equal(t, expectedNextRun, nextRun)
}

func TestTaskAtFuture(t *testing.T) {
	// Create new scheduler to have clean test env
	s := NewScheduler()

	now := time.Now()

	// Schedule to run in next minute
	nextMinuteTime := now.Add(time.Duration(1 * time.Minute))
	startAt := fmt.Sprintf("%02d:%02d", nextMinuteTime.Hour(), nextMinuteTime.Minute())
	dayJob := s.Every(1).Day().At(startAt)
	shouldBeFalse := false

	dayJob.Do(func() {
		shouldBeFalse = true
	})

	// Check first run
	expectedStartTime := time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), now.Add(time.Minute).Minute(), 0, 0, loc)
	nextRun := dayJob.NextScheduledTime()
	assert.Equal(t, expectedStartTime, nextRun)

	s.RunPending()

	// Check next run's scheduled time
	nextRun = dayJob.NextScheduledTime()
	assert.Equal(t, expectedStartTime, nextRun)
	assert.Equal(t, false, shouldBeFalse, "Day job was not expected to run as it was in the future")
=======
	assertEqualTime("next run", t, nextRun, startNext)
>>>>>>> prod-safety
}

func TestDaily(t *testing.T) {
	now := time.Now()

	// Create new scheduler to have clean test env
	s := NewScheduler()

	// schedule next run 1 day
<<<<<<< HEAD
	dayJob := s.Every(1).Day()
	dayJob.scheduleNextRun()
	tomorrow := now.AddDate(0, 0, 1)
	expectedTime := time.Date(tomorrow.Year(), tomorrow.Month(), tomorrow.Day(), 0, 0, 0, 0, loc)
	assert.Equal(t, expectedTime, dayJob.nextRun)

	// schedule next run 2 days
	dayJob = s.Every(2).Days()
	dayJob.scheduleNextRun()
	twoDaysFromNow := now.AddDate(0, 0, 2)
	expectedTime = time.Date(twoDaysFromNow.Year(), twoDaysFromNow.Month(), twoDaysFromNow.Day(), 0, 0, 0, 0, loc)
	assert.Equal(t, expectedTime, dayJob.nextRun)

	// Job running longer than next schedule 1day 2 hours
	dayJob = s.Every(1).Day()
	twoHoursFromNow := now.Add(time.Duration(2 * time.Hour))
	dayJob.lastRun = time.Date(twoHoursFromNow.Year(), twoHoursFromNow.Month(), twoHoursFromNow.Day(), twoHoursFromNow.Hour(), 0, 0, 0, loc)
	dayJob.scheduleNextRun()
	expectedTime = time.Date(now.Year(), now.Month(), now.AddDate(0, 0, 1).Day(), 0, 0, 0, 0, loc)
	assert.Equal(t, expectedTime, dayJob.nextRun)

	// At() 2 hours before now
	twoHoursBefore := now.Add(time.Duration(-2 * time.Hour))
	startAt := fmt.Sprintf("%02d:%02d", twoHoursBefore.Hour(), twoHoursBefore.Minute())
	dayJob = s.Every(1).Day().At(startAt)
	dayJob.scheduleNextRun()

	expectedTime = time.Date(twoHoursBefore.Year(), twoHoursBefore.Month(),
		twoHoursBefore.AddDate(0, 0, 1).Day(),
		twoHoursBefore.Hour(), twoHoursBefore.Minute(), 0, 0, loc)

	assert.Equal(t, expectedTime, dayJob.nextRun)
=======
	dayJob := s.Every(1, defaultOption).Day()
	dayJob.scheduleNextRun(true)
	exp := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, loc)
	assertEqualTime("1 day", t, dayJob.nextRun, exp)

	// schedule next run 2 days
	dayJob = s.Every(2, defaultOption).Days()
	dayJob.scheduleNextRun(true)
	exp = time.Date(now.Year(), now.Month(), now.Day()+2, 0, 0, 0, 0, loc)
	assertEqualTime("2 days", t, dayJob.nextRun, exp)

	// Job running longer than next schedule 1day 2 hours
	dayJob = s.Every(1, defaultOption).Day()
	dayJob.lastRun = time.Date(now.Year(), now.Month(), now.Day(), now.Hour()+2, 0, 0, 0, loc)
	dayJob.scheduleNextRun(true)
	exp = time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, loc)
	assertEqualTime("1 day 2 hours", t, dayJob.nextRun, exp)

	// At() 2 hours before now
	hour := now.Hour() - 2
	minute := now.Minute()
	startAt := fmt.Sprintf("%02d:%02d", hour, minute)
	dayJob = s.Every(1, defaultOption).Day().At(startAt)
	if err := dayJob.Err(); err != nil {
		t.Error(err)
	}

	dayJob.scheduleNextRun(true)
	exp = time.Date(now.Year(), now.Month(), now.Day()+1, hour, minute, 0, 0, loc)
	assertEqualTime("at 2 hours before now", t, dayJob.nextRun, exp)
>>>>>>> prod-safety
}

func TestWeekdayAfterToday(t *testing.T) {
	now := time.Now()

	// Create new scheduler to have clean test env
	s := NewScheduler()

	// Schedule job at next week day
	var weekJob *Job
	switch now.Weekday() {
	case time.Monday:
		weekJob = s.Every(1, defaultOption).Tuesday()
	case time.Tuesday:
		weekJob = s.Every(1, defaultOption).Wednesday()
	case time.Wednesday:
		weekJob = s.Every(1, defaultOption).Thursday()
	case time.Thursday:
		weekJob = s.Every(1, defaultOption).Friday()
	case time.Friday:
		weekJob = s.Every(1, defaultOption).Saturday()
	case time.Saturday:
		weekJob = s.Every(1, defaultOption).Sunday()
	case time.Sunday:
		weekJob = s.Every(1, defaultOption).Monday()
	}

	// First run
	weekJob.scheduleNextRun(true)
	exp := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, loc)
<<<<<<< HEAD
	assert.Equal(t, exp, weekJob.nextRun)
=======
	assertEqualTime("first run", t, weekJob.nextRun, exp)
>>>>>>> prod-safety

	// Simulate job run 7 days before
	weekJob.lastRun = weekJob.nextRun.AddDate(0, 0, -7)
	// Next run
	weekJob.scheduleNextRun(true)
	exp = time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, loc)
<<<<<<< HEAD
	assert.Equal(t, exp, weekJob.nextRun)
=======
	assertEqualTime("next run", t, weekJob.nextRun, exp)
>>>>>>> prod-safety
}

func TestWeekdayBeforeToday(t *testing.T) {
	now := time.Now()

	// Create new scheduler to have clean test env
	s := NewScheduler()

	// Schedule job at day before
	var weekJob *Job
	switch now.Weekday() {
	case time.Monday:
		weekJob = s.Every(1, defaultOption).Sunday()
	case time.Tuesday:
		weekJob = s.Every(1, defaultOption).Monday()
	case time.Wednesday:
		weekJob = s.Every(1, defaultOption).Tuesday()
	case time.Thursday:
		weekJob = s.Every(1, defaultOption).Wednesday()
	case time.Friday:
		weekJob = s.Every(1, defaultOption).Thursday()
	case time.Saturday:
		weekJob = s.Every(1, defaultOption).Friday()
	case time.Sunday:
		weekJob = s.Every(1, defaultOption).Saturday()
	}

<<<<<<< HEAD
	weekJob.scheduleNextRun()
	sixDaysFromNow := now.AddDate(0, 0, 6)

	exp := time.Date(sixDaysFromNow.Year(), sixDaysFromNow.Month(), sixDaysFromNow.Day(), 0, 0, 0, 0, loc)
	assert.Equal(t, exp, weekJob.nextRun)
=======
	weekJob.scheduleNextRun(true)
	exp := time.Date(now.Year(), now.Month(), now.Day()+6, 0, 0, 0, 0, loc)
	assertEqualTime("first run", t, weekJob.nextRun, exp)
>>>>>>> prod-safety

	// Simulate job run 7 days before
	weekJob.lastRun = weekJob.nextRun.AddDate(0, 0, -7)
	// Next run
<<<<<<< HEAD
	weekJob.scheduleNextRun()
	exp = time.Date(sixDaysFromNow.Year(), sixDaysFromNow.Month(), sixDaysFromNow.Day(), 0, 0, 0, 0, loc)
	assert.Equal(t, exp, weekJob.nextRun)
=======
	weekJob.scheduleNextRun(true)
	exp = time.Date(now.Year(), now.Month(), now.Day()+6, 0, 0, 0, 0, loc)
	assertEqualTime("nest run", t, weekJob.nextRun, exp)
>>>>>>> prod-safety
}

func TestWeekdayAt(t *testing.T) {
	now := time.Now()

	hour := now.Hour()
	minute := now.Minute()
	startAt := fmt.Sprintf("%02d:%02d", hour, minute)

	// Create new scheduler to have clean test env
	s := NewScheduler()

	// Schedule job at next week day
	var weekJob *Job
	switch now.Weekday() {
	case time.Monday:
		weekJob = s.Every(1, defaultOption).Tuesday().At(startAt)
		if err := weekJob.Err(); err != nil {
			t.Error(err)
		}
	case time.Tuesday:
		weekJob = s.Every(1, defaultOption).Wednesday().At(startAt)
		if err := weekJob.Err(); err != nil {
			t.Error(err)
		}
	case time.Wednesday:
		weekJob = s.Every(1, defaultOption).Thursday().At(startAt)
		if err := weekJob.Err(); err != nil {
			t.Error(err)
		}
	case time.Thursday:
		weekJob = s.Every(1, defaultOption).Friday().At(startAt)
		if err := weekJob.Err(); err != nil {
			t.Error(err)
		}
	case time.Friday:
		weekJob = s.Every(1, defaultOption).Saturday().At(startAt)
		if err := weekJob.Err(); err != nil {
			t.Error(err)
		}
	case time.Saturday:
		weekJob = s.Every(1, defaultOption).Sunday().At(startAt)
		if err := weekJob.Err(); err != nil {
			t.Error(err)
		}
	case time.Sunday:
		weekJob = s.Every(1, defaultOption).Monday().At(startAt)
		if err := weekJob.Err(); err != nil {
			t.Error(err)
		}
	}

	// First run
<<<<<<< HEAD
	weekJob.scheduleNextRun()
	exp := time.Date(now.Year(), now.Month(), now.AddDate(0, 0, 1).Day(), hour, minute, 0, 0, loc)
	assert.Equal(t, exp, weekJob.nextRun)
=======
	weekJob.scheduleNextRun(true)
	exp := time.Date(now.Year(), now.Month(), now.Day()+1, hour, minute, 0, 0, loc)
	assertEqualTime("first run", t, weekJob.nextRun, exp)
>>>>>>> prod-safety

	// Simulate job run 7 days before
	weekJob.lastRun = weekJob.nextRun.AddDate(0, 0, -7)
	// Next run
<<<<<<< HEAD
	weekJob.scheduleNextRun()
	exp = time.Date(now.Year(), now.Month(), now.AddDate(0, 0, 1).Day(), hour, minute, 0, 0, loc)
	assert.Equal(t, exp, weekJob.nextRun)
}

type lockerMock struct {
	cache map[string]struct{}
	l     sync.Mutex
}

func (l *lockerMock) Lock(key string) (bool, error) {
	l.l.Lock()
	defer l.l.Unlock()
	if _, ok := l.cache[key]; ok {
		return false, nil
	}
	l.cache[key] = struct{}{}
	return true, nil
}

func (l *lockerMock) Unlock(key string) error {
	l.l.Lock()
	defer l.l.Unlock()
	delete(l.cache, key)
	return nil
}

func TestSetLocker(t *testing.T) {
	if locker != nil {
		t.Fail()
		t.Log("Expected locker to not be set by default")
	}

	SetLocker(&lockerMock{})

	if locker == nil {
		t.Fail()
		t.Log("Expected locker to be set")
	}
}

type lockerResult struct {
	key   string
	cycle int
	s, e  time.Time
}

func TestLocker(t *testing.T) {
	l := sync.Mutex{}

	result := make([]lockerResult, 0)
	task := func(key string, i int) {
		s := time.Now()
		time.Sleep(time.Millisecond * 100)
		e := time.Now()
		l.Lock()
		result = append(result, lockerResult{
			key:   key,
			cycle: i,
			s:     s,
			e:     e,
		})
		l.Unlock()
	}

	SetLocker(&lockerMock{
		make(map[string]struct{}),
		sync.Mutex{},
	})

	for i := 0; i < 5; i++ {
		s1 := NewScheduler()
		s1.Every(1).Seconds().Lock().Do(task, "A", i)

		s2 := NewScheduler()
		s2.Every(1).Seconds().Lock().Do(task, "B", i)

		s3 := NewScheduler()
		s3.Every(1).Seconds().Lock().Do(task, "C", i)

		stop1 := s1.Start()
		stop2 := s2.Start()
		stop3 := s3.Start()

		time.Sleep(time.Millisecond * 100)

		close(stop1)
		close(stop2)
		close(stop3)

		for i := 0; i < len(result)-1; i++ {
			for j := i + 1; j < len(result); j++ {
				iBefJ := result[i].s.Before(result[j].s) && result[i].e.Before(result[j].s)
				jBefI := result[j].s.Before(result[i].s) && result[j].e.Before(result[i].s)
				if !iBefJ && !jBefI {
					t.Fatalf("\n2 operations ran concurrently:\n%s\n%d\n%s\n%s\n**********\n%s\n%d\n%s\n%s\n",
						result[i].key, result[i].cycle, result[i].s, result[i].e,
						result[j].key, result[j].cycle, result[j].s, result[j].e)
				}
			}
		}
	}
}

func TestGetAllJobs(t *testing.T) {
	defaultScheduler = NewScheduler()
	Every(1).Minute().Do(task)
	Every(2).Minutes().Do(task)
	Every(3).Minutes().Do(task)
	Every(4).Minutes().Do(task)
	js := Jobs()
	assert.Len(t, js, 4)
}

func TestTags(t *testing.T) {
	j := Every(1).Minute()
	j.Tag("some")
	j.Tag("tag")
	j.Tag("more")
	j.Tag("tags")

	assert.ElementsMatch(t, j.Tags(), []string{"tags", "tag", "more", "some"})

	j.Untag("more")
	assert.ElementsMatch(t, j.Tags(), []string{"tags", "tag", "some"})
}

func TestGetAt(t *testing.T) {
	j := Every(1).Minute().At("10:30")
	assert.Equal(t, "10:30", j.GetAt())
}

func TestGetWeekday(t *testing.T) {
	j := Every(1).Weekday(time.Wednesday)
	assert.Equal(t, time.Wednesday, j.GetWeekday())
=======
	weekJob.scheduleNextRun(true)
	exp = time.Date(now.Year(), now.Month(), now.Day()+1, hour, minute, 0, 0, loc)
	assertEqualTime("next run", t, weekJob.nextRun, exp)
}

type foo struct {
	jobNumber int64
}

func (f *foo) incr() {
	atomic.AddInt64(&f.jobNumber, 1)
}

func (f *foo) getN() int64 {
	return atomic.LoadInt64(&f.jobNumber)
}

const (
	expectedNumber       int64 = 10
	expectedNumberMinute int64 = 5
)

var (
	testF  *foo
	testF2 *foo
	client *redis.Client
)

func init() {
	s, err := miniredis.Run()
	if err != nil {
		log.Fatal(err)
	}
	// defer s.Close()

	client = redis.NewClient(&redis.Options{
		Addr: s.Addr(),
	})
	testF = new(foo)
	testF2 = new(foo)
}

func TestBasicDistributedJob1(t *testing.T) {
	t.Parallel()

	var defaultOption = func(j *Job) {
		j.DistributedJobName = "counter"
		j.DistributedRedisClient = client
	}

	sc := NewScheduler()
	sc.Every(1, defaultOption).Second().Do(testF.incr)

loop:
	for {
		select {
		case <-sc.Start():
		case <-time.After(10 * time.Second):
			sc.Clear()
			break loop
		}
	}

	if (expectedNumber-1 != testF.getN()) && (expectedNumber != testF.getN()) && (expectedNumber+1 != testF.getN()) {
		t.Errorf("1 expected number of jobs %d, got %d", expectedNumber, testF.getN())
	}

}

func TestBasicDistributedJob2(t *testing.T) {
	t.Parallel()

	var defaultOption = func(j *Job) {
		j.DistributedJobName = "counter"
		j.DistributedRedisClient = client
	}

	sc := NewScheduler()
	sc.Every(1, defaultOption).Second().Do(testF.incr)

loop:
	for {
		select {
		case <-sc.Start():
		case <-time.After(10 * time.Second):
			sc.Clear()
			break loop
		}
	}

	if (expectedNumber-1 != testF.getN()) && (expectedNumber != testF.getN()) && (expectedNumber+1 != testF.getN()) {
		t.Errorf("2 expected number of jobs %d, got %d", expectedNumber, testF.getN())
	}
}

func TestBasicDistributedJob3(t *testing.T) {
	t.Parallel()

	var defaultOption = func(j *Job) {
		j.DistributedJobName = "counter"
		j.DistributedRedisClient = client
	}

	sc := NewScheduler()
	sc.Every(1, defaultOption).Second().Do(testF.incr)

loop:
	for {
		select {
		case <-sc.Start():
		case <-time.After(10 * time.Second):
			sc.Clear()
			break loop
		}
	}

	if (expectedNumber-1 != testF.getN()) && (expectedNumber != testF.getN()) && (expectedNumber+1 != testF.getN()) {
		t.Errorf("3 expected number of jobs %d, got %d", expectedNumber, testF.getN())
	}
}

func TestBasicDistributedJob4(t *testing.T) {
	t.Parallel()

	var defaultOption = func(j *Job) {
		j.DistributedJobName = "counter"
		j.DistributedRedisClient = client
	}

	sc := NewScheduler()
	sc.Every(1, defaultOption).Second().Do(testF.incr)

loop:
	for {
		select {
		case <-sc.Start():
		case <-time.After(10 * time.Second):
			sc.Clear()
			break loop
		}
	}

	if (expectedNumber-1 != testF.getN()) && (expectedNumber != testF.getN()) && (expectedNumber+1 != testF.getN()) {
		t.Errorf("4 expected number of jobs %d, got %d", expectedNumber, testF.getN())
	}
}

func TestBasicDistributedJob5(t *testing.T) {
	t.Parallel()

	var defaultOption = func(j *Job) {
		j.DistributedJobName = "counter"
		j.DistributedRedisClient = client
	}

	sc := NewScheduler()
	sc.Every(1, defaultOption).Second().Do(testF.incr)

loop:
	for {
		select {
		case <-sc.Start():
		case <-time.After(10 * time.Second):
			sc.Clear()
			break loop
		}
	}

	if (expectedNumber-1 != testF.getN()) && (expectedNumber != testF.getN()) && (expectedNumber+1 != testF.getN()) {
		t.Errorf("5 expected number of jobs %d, got %d", expectedNumber, testF.getN())
	}
}

func TestBasicDistributedJobMinute1(t *testing.T) {
	if t.Skipped() {
		return
	}

	t.Parallel()
	var defaultOption = func(j *Job) {
		j.DistributedJobName = "counter"
		j.DistributedRedisClient = client
	}

	sc := NewScheduler()
	sc.Every(1, defaultOption).Minute().Do(testF2.incr)

loop:
	for {
		select {
		case <-sc.Start():
		case <-time.After(60 * time.Second):
			sc.Clear()
			break loop
		}
	}

	if (expectedNumberMinute-1 != testF2.getN()) && (expectedNumberMinute != testF2.getN()) && (expectedNumberMinute+1 != testF2.getN()) {
		t.Errorf("1 expected number of jobs %d, got %d", expectedNumberMinute, testF2.getN())
	}

}

func TestBasicDistributedJobMinute2(t *testing.T) {
	if t.Skipped() {
		return
	}

	t.Parallel()
	var defaultOption = func(j *Job) {
		j.DistributedJobName = "counter"
		j.DistributedRedisClient = client
	}

	sc := NewScheduler()
	sc.Every(1, defaultOption).Minute().Do(testF2.incr)

loop:
	for {
		select {
		case <-sc.Start():
		case <-time.After(60 * time.Second):
			sc.Clear()
			break loop
		}
	}

	if (expectedNumberMinute-1 != testF2.getN()) && (expectedNumberMinute != testF2.getN()) && (expectedNumberMinute+1 != testF2.getN()) {
		t.Errorf("1 expected number of jobs %d, got %d", expectedNumberMinute, testF2.getN())
	}
}

func TestBasicDistributedJobMinute3(t *testing.T) {
	if t.Skipped() {
		return
	}

	t.Parallel()
	var defaultOption = func(j *Job) {
		j.DistributedJobName = "counter"
		j.DistributedRedisClient = client
	}

	sc := NewScheduler()
	sc.Every(1, defaultOption).Minute().Do(testF2.incr)

loop:
	for {
		select {
		case <-sc.Start():
		case <-time.After(60 * time.Second):
			sc.Clear()
			break loop
		}
	}

	if (expectedNumberMinute-1 != testF2.getN()) && (expectedNumberMinute != testF2.getN()) && (expectedNumberMinute+1 != testF2.getN()) {
		t.Errorf("1 expected number of jobs %d, got %d", expectedNumberMinute, testF2.getN())
	}
}

func TestBasicDistributedJobMinute4(t *testing.T) {
	if t.Skipped() {
		return
	}

	t.Parallel()
	var defaultOption = func(j *Job) {
		j.DistributedJobName = "counter"
		j.DistributedRedisClient = client
	}

	sc := NewScheduler()
	sc.Every(1, defaultOption).Minute().Do(testF2.incr)

loop:
	for {
		select {
		case <-sc.Start():
		case <-time.After(60 * time.Second):
			sc.Clear()
			break loop
		}
	}

	if (expectedNumberMinute-1 != testF2.getN()) && (expectedNumberMinute != testF2.getN()) && (expectedNumberMinute+1 != testF2.getN()) {
		t.Errorf("1 expected number of jobs %d, got %d", expectedNumberMinute, testF2.getN())
	}
}

func TestBasicDistributedJobMinute5(t *testing.T) {
	if t.Skipped() {
		return
	}

	t.Parallel()
	var defaultOption = func(j *Job) {
		j.DistributedJobName = "counter"
		j.DistributedRedisClient = client
	}

	sc := NewScheduler()
	sc.Every(1, defaultOption).Minute().Do(testF2.incr)

loop:
	for {
		select {
		case <-sc.Start():
		case <-time.After(60 * time.Second):
			sc.Clear()
			break loop
		}
	}

	if (expectedNumberMinute-1 != testF2.getN()) && (expectedNumberMinute != testF2.getN()) && (expectedNumberMinute+1 != testF2.getN()) {
		t.Errorf("1 expected number of jobs %d, got %d", expectedNumberMinute, testF2.getN())
	}
>>>>>>> prod-safety
}
