package sky

// utcToTAI is an instant on utc told on tai: the seconds into its utc
// day, and that day's leap seconds, from the start of the same day.
func (moment Instant) utcToTAI() (*Instant, error) {
	ahead, err := aheadOn(moment.day)
	if err != nil {
		return nil, err
	}
	seconds := moment.part * dayLength(UTC(), moment.day)
	atomic := settled(TAI(), moment.day, (seconds+ahead)/secondsADay)
	return &atomic, nil
}

// The tries taiToUTC makes, and what it misses by when it is done.
const (
	// tries is defined by sofa's taiutc, which tries three times.
	tries = 3
	// missed is defined: a utc instant that is the tai one misses by
	// nothing.
	missed = 0.0
)

// taiToUTC is an instant on tai told on utc, by trying the same day and
// fraction on utc and moving it by what it misses by, as sofa's taiutc
// does, until it misses by nothing, three tries at most.
func (atomic Instant) taiToUTC() (*Instant, error) {
	civil := atomic.relabel(UTC())
	for range tries {
		back, err := civil.utcToTAI()
		if err != nil {
			return nil, err
		}
		apart := ((float64(atomic.day) - float64(back.day)) +
			(atomic.part - back.part)) * secondsADay
		if apart == missed {
			break
		}
		civil = civil.utcAfter(apart)
	}
	return &civil, nil
}

// utcAfter is an instant on utc moved so many seconds, over days that may
// be 86401 s long.
func (moment Instant) utcAfter(seconds float64) Instant {
	day := moment.day
	into := moment.part*dayLength(UTC(), day) + seconds
	for into < missed {
		day--
		into += dayLength(UTC(), day)
	}
	for into >= dayLength(UTC(), day) {
		into -= dayLength(UTC(), day)
		day++
	}
	return Instant{scale: UTC(), day: day,
		part: into / dayLength(UTC(), day)}
}
