package mediasoup

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

var (
	audioLevelRouter      *Router
	audioLevelObserver    RtpObserver
	audioLevelMediaCodecs = []RtpCodecCapability{
		{
			Kind:      "audio",
			MimeType:  "audio/opus",
			ClockRate: 48000,
			Channels:  2,
			Parameters: &RtpParameter{
				Useinbandfec: 1,
			},
		},
	}
)

func init() {
	AddBeforeEach(func() {
		router, err := worker.CreateRouter(audioLevelMediaCodecs)
		if err != nil {
			panic(err)
		}
		audioLevelRouter = router
	})
}

func TestCreateAudioLevelObserver_Succeeds(t *testing.T) {
	audioLevelObserver, err := audioLevelRouter.CreateAudioLevelObserver(nil)

	assert.NoError(t, err)
	assert.False(t, audioLevelObserver.Closed())
	assert.False(t, audioLevelObserver.Paused())

	dump := audioLevelRouter.Dump()

	var result struct {
		RtpObserverIds []string
	}
	assert.NoError(t, dump.Result(&result))
	assert.Equal(t, []string{audioLevelObserver.Id()}, result.RtpObserverIds)
}

func TestCreateAudioLevelObserver_TypeError(t *testing.T) {
	_, err := audioLevelRouter.CreateAudioLevelObserver(&CreateAudioLevelObserverParams{
		MaxEntries: 0,
	})
	assert.IsType(t, err, NewTypeError(""))
}

func TestCreateAudioLevelObserver_Pause_Resume(t *testing.T) {
	audioLevelObserver, err := audioLevelRouter.CreateAudioLevelObserver(nil)

	assert.NoError(t, err)

	audioLevelObserver.Pause()

	assert.True(t, audioLevelObserver.Paused())

	audioLevelObserver.Resume()

	assert.False(t, audioLevelObserver.Paused())
}

func TestCreateAudioLevelObserver_Close(t *testing.T) {
	_, err := audioLevelRouter.CreateAudioLevelObserver(nil)
	assert.NoError(t, err)
	audioLevelObserver2, err := audioLevelRouter.CreateAudioLevelObserver(nil)
	assert.NoError(t, err)

	dump := audioLevelRouter.Dump()
	var result struct {
		RtpObserverIds []string
	}
	assert.NoError(t, dump.Result(&result))

	assert.Equal(t, 2, len(result.RtpObserverIds))

	audioLevelObserver2.Close()

	assert.True(t, audioLevelObserver2.Closed())

	dump = audioLevelRouter.Dump()
	assert.NoError(t, dump.Result(&result))

	assert.Equal(t, 1, len(result.RtpObserverIds))
}

func TestCreateAudioLevelObserver_Router_Close(t *testing.T) {
	audioLevelObserver, err := audioLevelRouter.CreateAudioLevelObserver(nil)
	assert.NoError(t, err)

	routerclose := false
	audioLevelObserver.On("routerclose", func() {
		routerclose = true
	})
	audioLevelRouter.Close()

	assert.True(t, audioLevelObserver.Closed())
	assert.True(t, routerclose)
}

func TestCreateAudioLevelObserver_Worker_Close(t *testing.T) {
	worker, err := NewWorker("")
	assert.NoError(t, err)

	router, err := worker.CreateRouter(audioLevelMediaCodecs)
	assert.NoError(t, err)

	audioLevelObserver, err := router.CreateAudioLevelObserver(nil)
	assert.NoError(t, err)

	routerclose := false
	audioLevelObserver.On("routerclose", func() {
		routerclose = true
	})
	worker.Close()

	assert.True(t, audioLevelObserver.Closed())
	assert.True(t, routerclose)
}
