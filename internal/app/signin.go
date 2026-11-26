package app

import (
	"context"
	"errors"
	"yoho-music/internal/i18n"

	"yoho-music/internal/music"
	"yoho-music/internal/plugin"
)

type pluginAccount struct {
	source  music.Source
	account *music.Account
	busy    bool
}

// Account results belong to the plugin instance that initiated them, so a
// replaced plugin cannot refresh the newly installed instance's account.
func (m *model) setAccount(id string, account pluginAccount) {
	if m.sourceByID(id) != account.source {
		return
	}
	previous := m.accounts.GetUntracked()[id]
	previousID, accountID := "", ""
	if previous.account != nil {
		previousID = previous.account.ID
	}
	if account.account != nil {
		accountID = account.account.ID
	}
	if previous.source != account.source || previousID != accountID {
		for key := range m.savedCollections {
			if key.source.Info().ID == id {
				delete(m.savedCollections, key)
			}
		}
		for key := range m.likes {
			if key.source.Info().ID == id {
				delete(m.likes, key)
			}
		}
	}
	m.accounts.Update(func(current map[string]pluginAccount) map[string]pluginAccount {
		updated := make(map[string]pluginAccount, len(current)+1)
		for key, value := range current {
			updated[key] = value
		}
		updated[id] = account
		return updated
	})
}

func (m *model) accountOf(source music.Source) pluginAccount {
	state := m.accounts.Get()[source.Info().ID]
	if state.source != source {
		return pluginAccount{source: source}
	}
	return state
}

func (m *model) refreshAccount(source music.Source) {
	if !source.Can("account") || m.accountOf(source).busy {
		return
	}
	id := source.Info().ID
	state := m.accountOf(source)
	state.busy = true
	m.setAccount(id, state)
	go func() {
		account, _ := source.Account()
		m.cx.Post(func() { m.setAccount(id, pluginAccount{source: source, account: account}) })
	}()
}

func (m *model) runAccount(source music.Source, signingIn bool, attempt context.Context) {
	id := source.Info().ID
	state := m.accountOf(source)
	if state.busy {
		return
	}
	m.error.Set("")
	state.busy = true
	m.setAccount(id, state)
	go func() {
		account := state.account
		var err error
		if signingIn {
			var signedIn *music.Account
			if p, ok := source.(*plugin.NSGoSource); ok {
				signedIn, err = p.SignInWithPrompt(attempt, func(prompt music.SignInPrompt) {
					dialog := m.signInDialog.GetUntracked()
					if dialog.attempt == attempt && dialog.prompt != prompt {
						dialog.prompt = prompt
						m.signInDialog.Set(dialog)
					}
				})
			} else {
				signedIn, err = source.SignIn()
			}
			if err == nil {
				account = signedIn
			}
		} else if err = source.SignOut(); err == nil {
			account = nil
		}
		m.cx.Post(func() {
			if m.sourceByID(id) != source {
				return
			}
			m.setAccount(id, pluginAccount{source: source, account: account})
			if err == nil {
				m.sourceVersion.Update(func(n int) int { return n + 1 })
			}
			dialog := m.signInDialog.GetUntracked()
			if signingIn && dialog.attempt == attempt {
				canceled := errors.Is(err, context.Canceled)
				if err == nil || canceled {
					m.closeSignInDialog()
					return
				}
				dialog.cancel()
				dialog.attempt, dialog.cancel = nil, nil
				dialog.prompt = music.SignInPrompt{}
				dialog.problem = err.Error()
				if errors.Is(err, context.DeadlineExceeded) || err.Error() == context.DeadlineExceeded.Error() {
					dialog.problem = i18n.T("Sign-in timed out. Try again.")
				}
				m.signInDialog.Set(dialog)
			} else if !signingIn {
				m.report(err)
			}
		})
	}()
}
