package main

import (
	"flag"
	"fmt"
	"image/color"
	"log"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"streaming-audio-redes/internal/server"
	"streaming-audio-redes/internal/ui"
)

func formatTime(sec float64) string {
	if sec < 0 || math.IsNaN(sec) {
		sec = 0
	}
	m := int(sec) / 60
	s := int(sec) % 60
	return fmt.Sprintf("%d:%02d", m, s)
}

func main() {
	addr := flag.String("addr", ":9090", "endereço TCP da rádio")
	inputDevice := flag.String("input", "", "microfone (substring do nome; vazio = padrão)")
	sampleRate := flag.Uint("sample-rate", 44100, "taxa de amostragem (Hz)")
	channels := flag.Uint("channels", 2, "canais (2=stéreo, igual aos WAV em assets/)")
	flag.Parse()

	studio := server.NewStudio(server.StudioConfig{
		Live: server.LiveConfig{
			DeviceName: *inputDevice,
			SampleRate: uint32(*sampleRate),
			Channels:   uint16(*channels),
		},
	})

	for _, name := range []string{"musica.wav", "musica2.wav"} {
		path := filepath.Join("assets", name)
		if _, err := os.Stat(path); err != nil {
			continue
		}
		if err := studio.AddTrack(path); err != nil {
			log.Printf("[estúdio] não foi possível pré-carregar %s: %v", name, err)
		}
	}

	radio := server.New(*addr, studio)
	studio.SetMetaChangeHandler(radio.BroadcastMeta)

	go func() {
		if err := radio.Run(); err != nil {
			log.Printf("[estúdio] servidor encerrado: %v", err)
		}
	}()

	a := app.NewWithID("br.ufes.back2past.studio")
	ui.ApplyTheme(a)
	w := a.NewWindow("Back2Past — Cabine do Locutor")
	w.Resize(ui.StudioWindowSize())

	onAirStatus := canvas.NewText("Transmissão pausada", ui.ColorMuted)
	onAirStatus.TextSize = 11

	listenersLabel := canvas.NewText("Ouvintes: 0", ui.ColorForeground)
	listenersLabel.TextSize = 12
	listenersLabel.TextStyle = fyne.TextStyle{Monospace: true}

	onAirBadge := ui.NewOnAirBadge()

	trackCaption := ui.MonoCaption("Faixa atual")
	trackTitle := canvas.NewText("Nenhuma faixa no ar", ui.ColorForeground)
	trackTitle.TextSize = 17
	trackTitle.TextStyle = fyne.TextStyle{Bold: true}

	micToggle := widget.NewCheck("Microfone aberto", func(on bool) {
		studio.SetMicEnabled(on)
	})

	micVolSlider := widget.NewSlider(0, 100)
	micVolSlider.SetValue(100)
	micVolLabel := ui.MonoCaption("Voz")
	micVolValue := canvas.NewText("100%", ui.ColorMuted)
	micVolValue.TextSize = 10
	micVolValue.TextStyle = fyne.TextStyle{Monospace: true}
	micVolSlider.OnChanged = func(v float64) {
		studio.SetMicVolume(float32(v / 100))
		micVolValue.Text = strconv.Itoa(int(v)) + "%"
		canvas.Refresh(micVolValue)
	}

	musicVolSlider := widget.NewSlider(0, 100)
	musicVolSlider.SetValue(75)
	musicVolLabel := ui.MonoCaption("Música")
	musicVolValue := canvas.NewText("75%", ui.ColorMuted)
	musicVolValue.TextSize = 10
	musicVolValue.TextStyle = fyne.TextStyle{Monospace: true}
	musicVolSlider.OnChanged = func(v float64) {
		studio.SetMusicVolume(float32(v / 100))
		musicVolValue.Text = strconv.Itoa(int(v)) + "%"
		canvas.Refresh(musicVolValue)
	}

	vuMeter := ui.NewVuMeter()

	var paths []string
	playlistList := widget.NewList(
		func() int { return len(paths) },
		func() fyne.CanvasObject { return widget.NewLabel("") },
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			if int(id) >= len(paths) {
				return
			}
			obj.(*widget.Label).SetText(filepath.Base(paths[id]))
		},
	)

	refreshPlaylist := func() {
		paths = studio.Playlist()
		playlistList.Refresh()
	}

	selected := -1
	playlistList.OnSelected = func(id widget.ListItemID) { selected = int(id) }
	playlistList.OnUnselected = func(widget.ListItemID) { selected = -1 }

	showError := func(err error) {
		if err != nil {
			dialog.ShowError(err, w)
		}
	}

	var updatingTimeline bool
	timeSlider := widget.NewSlider(0, 1)
	timeSlider.Step = 0.1
	timeSlider.Disable()
	currentTimeLabel := canvas.NewText("0:00", ui.ColorPrimary)
	currentTimeLabel.TextSize = 12
	currentTimeLabel.TextStyle = fyne.TextStyle{Monospace: true}
	durationLabel := canvas.NewText("0:00", ui.ColorMuted)
	durationLabel.TextSize = 12
	durationLabel.TextStyle = fyne.TextStyle{Monospace: true}

	timeSlider.OnChanged = func(v float64) {
		if updatingTimeline {
			return
		}
		studio.SeekMusicTo(v)
	}

	playBtn := widget.NewButton("▶ Tocar", func() {
		if selected < 0 || selected >= len(paths) {
			dialog.ShowInformation("Playlist", "Selecione uma faixa na lista.", w)
			return
		}
		showError(studio.PlayTrack(paths[selected]))
	})
	stopMusicBtn := widget.NewButton("■ Parar", func() { studio.StopMusic() })
	seekFwdBtn := widget.NewButton("+10s", func() { studio.SeekMusicForward(10) })
	silenceBtn := widget.NewButton("Fora do ar", func() {
		micToggle.SetChecked(false)
		studio.StopOnAir()
	})

	addBtn := widget.NewButton("Adicionar música…", func() {
		dialog.ShowFileOpen(func(reader fyne.URIReadCloser, err error) {
			if err != nil || reader == nil {
				return
			}
			defer reader.Close()
			showError(studio.AddTrack(reader.URI().Path()))
			fyne.Do(refreshPlaylist)
		}, w)
	})
	removeBtn := widget.NewButton("Remover", func() {
		if selected < 0 || selected >= len(paths) {
			return
		}
		studio.RemoveTrack(paths[selected])
		selected = -1
		refreshPlaylist()
	})

	refreshTimeline := func(snap server.StudioSnapshot) {
		updatingTimeline = true
		defer func() { updatingTimeline = false }()

		if snap.MusicPlaying && snap.MusicDurationSec > 0 {
			timeSlider.Max = snap.MusicDurationSec
			timeSlider.SetValue(snap.MusicPositionSec)
			timeSlider.Enable()
			trackTitle.Text = filepath.Base(snap.MusicPath)
		} else {
			timeSlider.SetValue(0)
			timeSlider.Max = 1
			timeSlider.Disable()
			trackTitle.Text = "Nenhuma faixa no ar"
		}
		currentTimeLabel.Text = formatTime(snap.MusicPositionSec)
		durationLabel.Text = formatTime(snap.MusicDurationSec)
		canvas.Refresh(trackTitle)
		canvas.Refresh(currentTimeLabel)
		canvas.Refresh(durationLabel)
	}

	refreshStatus := func() {
		snap := studio.Snapshot()
		listeners, packets := radio.Stats()
		onAir := snap.OnAir != "Silêncio"
		ui.SetOnAir(onAirBadge, onAir)
		vuMeter.SetActive(onAir && (snap.MicEnabled || snap.MusicPlaying))

		if onAir {
			onAirStatus.Text = "No ar: " + snap.OnAir
		} else {
			onAirStatus.Text = "Transmissão pausada — ouvintes sem sinal"
		}
		canvas.Refresh(onAirStatus)

		listenersLabel.Text = fmt.Sprintf("Ouvintes: %d · Pacotes: %d · %s", listeners, packets, *addr)
		canvas.Refresh(listenersLabel)

		if micToggle.Checked != snap.MicEnabled {
			micToggle.SetChecked(snap.MicEnabled)
		}
		refreshTimeline(snap)
	}

	go func() {
		for range time.Tick(200 * time.Millisecond) {
			fyne.Do(refreshStatus)
		}
	}()

	timelineRow := container.NewBorder(
		nil, nil,
		currentTimeLabel,
		durationLabel,
		timeSlider,
	)

	signalRow := container.NewBorder(nil, nil,
		ui.SpacedVBox(4, ui.MonoCaption("Sinal de transmissão"), onAirStatus),
		onAirBadge,
		nil,
	)
	broadcastPanel := ui.DisplayPanel(ui.SpacedVBox(6, signalRow, micToggle))

	trackPanel := ui.DisplayPanel(ui.SpacedVBox(6,
		trackCaption,
		trackTitle,
		timelineRow,
		vuMeter,
		container.NewGridWithColumns(4, playBtn, stopMusicBtn, seekFwdBtn, silenceBtn),
	))

	mixPanel := ui.DisplayPanel(ui.SpacedVBox(6,
		ui.MonoCaption("Mixagem"),
		container.NewBorder(nil, nil, micVolLabel, micVolValue, micVolSlider),
		container.NewBorder(nil, nil, musicVolLabel, musicVolValue, musicVolSlider),
	))

	playlistHeader := ui.SpacedVBox(6,
		ui.MonoCaption("Playlist"),
		container.NewGridWithColumns(2, addBtn, removeBtn),
	)
	playlistMin := canvas.NewRectangle(color.Transparent)
	playlistMin.SetMinSize(fyne.NewSize(0, 96))
	playlistBody := container.NewStack(playlistMin, playlistList)
	playlistPanel := ui.DisplayPanel(container.NewBorder(playlistHeader, nil, nil, nil, playlistBody))

	header := ui.BrandHeader("Cabine do Locutor", "Painel de controle ao vivo", true)

	content := ui.SpacedVBox(ui.SectionGap,
		header,
		broadcastPanel,
		trackPanel,
		mixPanel,
		playlistPanel,
		listenersLabel,
		ui.Footer(),
	)

	scroll := container.NewScroll(ui.CardWrap(content))
	scroll.SetMinSize(fyne.NewSize(ui.StudioWindowSize().Width, 0))
	w.SetContent(scroll)
	w.CenterOnScreen()
	refreshPlaylist()
	refreshStatus()
	w.ShowAndRun()
}
