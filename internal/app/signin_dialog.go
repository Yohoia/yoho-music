package app

import (
	"context"
	"yoho-music/internal/i18n"

	"github.com/nsgo-dev/nsgo/base"
	"github.com/nsgo-dev/nsgo/core"
	"github.com/nsgo-dev/nsgo/icons/lucide"
	"rsc.io/qr"
	"yoho-music/internal/music"
)

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
	m.closeSignInDialog()
	m.signInDialog.Set(signInDialogState{source: source})
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
		message := i18n.T("The plugin will guide you through signing in.")
		switch {
		case state.problem != "":
			message = state.problem
		case state.prompt.Text != "":
			message = state.prompt.Text
		case state.attempt != nil:
			message = i18n.T("Waiting for you to finish signing in…")
		}
		children := []core.Node{
			label(title(), 18, core.FontWeightBold, p.text),
			label(message, 13, core.FontWeightNormal, p.muted),
		}
		if state.prompt.QR != "" {
			code, err := qr.Encode(state.prompt.QR, qr.M)
			if err == nil {
				children = append(children, core.View{Style: core.Style{AlignItems: core.AlignCenter}, Children: []core.Node{signInQR(code)}})
			}
		}
		actions := row(spacer(), button(i18n.T("Cancel"), "sign-in-cancel", lucide.X, m.closeSignInDialog))
		if state.attempt == nil {
			actions.Children = append(actions.Children, button(i18n.T("Sign In"), "sign-in-confirm", lucide.LogIn, func() {
				current := m.signInDialog.GetUntracked()
				if current.source != state.source || current.attempt != nil || m.accountOf(state.source).busy {
					return
				}
				current.attempt, current.cancel = context.WithCancel(m.context)
				current.problem = ""
				m.signInDialog.Set(current)
				m.runAccount(state.source, true, current.attempt)
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
		LabelFunc: title, TestID: "sign-in-dialog",
		BackdropStyle: core.Style{Background: core.RGBA(0, 0.35)},
		PopupStyle:    core.Style{Width: core.Pt(360), Padding: core.All(24), CornerRadius: 18, Background: p.background},
		Children:      []core.Node{body}}
}

func signInQR(code *qr.Code) core.Node {
	const size = 216
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
