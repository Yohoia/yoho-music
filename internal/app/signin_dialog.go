package app

import (
	"context"

	"github.com/nsgo-dev/nsgo/base"
	"github.com/nsgo-dev/nsgo/core"
	"github.com/nsgo-dev/nsgo/icons/lucide"
	"github.com/nsgo-dev/nsgo/ui"
	"rsc.io/qr"
	"yoho-music/internal/i18n"
	"yoho-music/internal/music"
	"yoho-music/internal/plugin"
)

const signInQRSize = 216

type signInDialogState struct {
	source  music.Source
	attempt context.Context
	cancel  context.CancelFunc
	prompt  music.SignInPrompt
	problem string
}

func (m *model) changeAccount(source music.Source, signingIn bool) {
	if m.accountOf(source).busy {
		return
	}
	if !signingIn {
		m.runAccount(source, false, nil)
		return
	}
	if current := m.signInDialog.GetUntracked(); current.cancel != nil {
		current.cancel()
	}
	attempt, cancel := context.WithCancel(m.context)
	m.signInDialog.Set(signInDialogState{source: source, attempt: attempt, cancel: cancel})
	m.runAccount(source, true, attempt)
}

func (m *model) closeSignInDialog() {
	if cancel := m.signInDialog.GetUntracked().cancel; cancel != nil {
		cancel()
	}
	m.signInDialog.Set(signInDialogState{})
}

func (m *model) signInDialogView(cx *core.Scope) core.Node {
	p := colors(cx)
	title := func() string {
		state := m.signInDialog.Get()
		if state.source == nil {
			return ""
		}
		return i18n.T("Sign in to ") + state.source.Info().Name
	}
	body := core.Dynamic(func(cx *core.Scope) core.Node {
		state := m.signInDialog.Get()
		if state.source == nil {
			return nil
		}
		message := i18n.T("Starting sign-in…")
		switch {
		case state.problem != "":
			message = state.problem
		case state.prompt.Text != "":
			message = state.prompt.Text
		}
		children := []core.Node{
			label(title(), 18, core.FontWeightBold, p.text),
			label(message, 13, core.FontWeightNormal, p.muted),
		}
		var artwork core.Node
		if state.prompt.QR != "" {
			code, err := qr.Encode(state.prompt.QR, qr.M)
			if err == nil {
				artwork = signInQR(code)
			}
		}
		if artwork == nil && state.attempt != nil && state.source.Info().ID == plugin.NetEaseSourceID {
			artwork = ui.Spinner{Variant: ui.SpinnerSpokes, Size: 24, Label: message}
		}
		if artwork != nil {
			children = append(children, core.View{Style: core.Style{Height: core.Pt(signInQRSize), AlignItems: core.AlignCenter, JustifyContent: core.JustifyCenter},
				Children: []core.Node{artwork}})
		}
		cancel := button(i18n.T("Cancel"), "sign-in-cancel", lucide.X, m.closeSignInDialog)
		cancel.Variant = ui.ButtonOutline
		actions := row(spacer(), cancel)
		if state.problem != "" {
			actions.Children = append(actions.Children, button(i18n.T("Retry"), "sign-in-retry", lucide.RefreshCw, func() {
				m.changeAccount(state.source, true)
			}))
		}
		children = append(children, actions)
		return core.View{Style: core.Style{Gap: 12}, Children: children}
	})
	return base.Dialog{Open: func() bool { return m.signInDialog.Get().source != nil },
		OnOpenChange: func(open bool, _ base.Reason) {
			if !open {
				m.closeSignInDialog()
			}
		},
		LabelFunc: title, TestID: "sign-in-dialog", Animation: core.ModalFade,
		BackdropStyle: core.Style{Background: core.RGBA(0, 0.35)},
		PopupStyle:    core.Style{Width: core.Pt(360), Padding: core.All(24), CornerRadius: 18, Background: p.background},
		Children:      []core.Node{body}}
}

func signInQR(code *qr.Code) core.Node {
	const size = signInQRSize
	return core.Canvas{TestID: "sign-in-qr", Label: i18n.T("Sign-in QR Code"), Style: core.Style{Width: core.Pt(size), Height: core.Pt(size)}, Draw: func(d *core.Drawing, _ core.Size) {
		cell := float32(size / (code.Size + 8))
		offset := (size - float32(code.Size)*cell) / 2
		d.SetFillColor(core.RGB(0xffffff)).FillRect(0, 0, size, size)
		d.SetFillColor(core.RGB(0x141416))
		for y := range code.Size {
			for x := range code.Size {
				if code.Black(x, y) {
					d.FillRect(offset+float32(x)*cell, offset+float32(y)*cell, cell, cell)
				}
			}
		}
	}}
}
