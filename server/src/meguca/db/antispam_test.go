package db

import (
	"meguca/auth"
	"meguca/common"
	"meguca/config"
	. "meguca/test"
	"testing"
	"time"

	"github.com/lib/pq"
)

func TestSpamScores(t *testing.T) {
	config.Set(config.Configs{
		CaptchaTags: config.Defaults.CaptchaTags,
		Public: config.Public{
			Captcha: true,
		},
	})
	assertTableClear(t, "spam_scores", "last_solved_captchas")
	err := auth.LoadCaptchaServices()
	if err != nil {
		t.Fatal(err)
	}
	spamDetection := newListener(t, "spam_detected")
	defer spamDetection.Close()
	now := time.Now().Round(time.Second)

	for _, ip := range [...]string{
		"226.209.126.221",
		"131.215.1.14",
		"99.188.17.210",
		"71.189.25.162",
	} {
		err := ValidateCaptcha(auth.CreateTestCaptcha(t), ip)
		if err != nil {
			t.Fatal(err)
		}
	}

	for ip, score := range map[string]int64{
		"131.215.1.14":  now.Add(-20 * spamDetectionThreshold).Unix(),
		"99.188.17.210": now.Add(-5 * time.Second).Unix(),
		"71.189.25.162": now.Add(10 * spamDetectionThreshold).Unix(),
	} {
		_, err = sq.Insert("spam_scores").
			Columns("ip", "score").
			Values(ip, score).
			Exec()
		if err != nil {
			t.Fatal(err)
		}
	}

	spamMu.Lock()
	spamScoreBuffer = map[string]time.Duration{
		"226.209.126.221": time.Second * 10,
		"131.215.1.14":    time.Second * 10,
		"99.188.17.210":   time.Second * 10,
		"71.189.25.162":   spamDetectionThreshold,
	}
	err = flushSpamScores()
	spamMu.Unlock()
	if err != nil {
		t.Fatal(err)
	}

	cases := [...]struct {
		name, ip       string
		needCaptcha    bool
		needCaptchaErr error
	}{
		{"fresh write", "226.209.126.221", false, nil},
		{"overwrite stale value", "131.215.1.14", false, nil},
		{"increment DB value", "99.188.17.210", true, nil},
		{"spam", "71.189.25.162", true, common.ErrSpamDected},
		{"no captcha solved in 3h", "143.195.24.54", true, nil},
	}

	for i := range cases {
		c := cases[i]
		t.Run(c.name, func(t *testing.T) {
			need, err := NeedCaptcha(c.ip)
			if err != c.needCaptchaErr {
				UnexpectedError(t, err)
			}
			AssertDeepEquals(t, need, c.needCaptcha)
		})
	}

	t.Run("spam detection propagation", func(t *testing.T) {
		msg := <-spamDetection.Notify
		AssertDeepEquals(t, msg.Extra, "71.189.25.162")
	})

	t.Run("clear score", func(t *testing.T) {
		const ip = "99.188.17.210"
		err := resetSpamScore(ip)
		if err != nil {
			t.Fatal(err)
		}
		need, err := NeedCaptcha(ip)
		if err != nil {
			t.Fatal(err)
		}
		AssertDeepEquals(t, need, false)
	})

	t.Run("expiry", func(t *testing.T) {
		err := expireSpamScores()
		if err != nil {
			t.Fatal(err)
		}
	})
}

func newListener(t *testing.T, channel string) *pq.Listener {
	t.Helper()
	l := pq.NewListener(
		TestConnArgs,
		time.Second,
		time.Second*10,
		func(_ pq.ListenerEventType, _ error) {},
	)
	err := l.Listen(channel)
	if err != nil {
		t.Fatal(err)
	}
	return l
}
