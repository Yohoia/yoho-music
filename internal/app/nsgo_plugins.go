package app

import (
	"slices"
	"strings"
	"yoho-music/internal/i18n"

	"github.com/nsgo-dev/nsgo/base"
	"github.com/nsgo-dev/nsgo/core"
	"github.com/nsgo-dev/nsgo/icons/lucide"
	"yoho-music/internal/local"
	"yoho-music/internal/plugin"
)

func (m *model) pickNSGoPlugin(cx *core.Scope) {
	store := m.nsgoStore.GetUntracked()
	if store == nil || m.nsgoPluginBusy.GetUntracked() {
		return
	}
	cx.PickFiles(core.PickFilesOptions{Types: []string{".nsgopkg"}}, func(files []core.File, err error) {
		m.report(err)
		if err != nil || len(files) == 0 || m.nsgoPluginBusy.GetUntracked() {
			return
		}
		go func(path string) {
			p, err := store.ReadInstall(path)
			cx.Post(func() {
				if err != nil {
					m.report(err)
					return
				}
				m.confirmPluginInstall(cx, store, p)
			})
		}(files[0].Path)
	})
}

func (m *model) confirmPluginInstall(cx *core.Scope, store *plugin.NSGoStore, p *plugin.NSGoPackage) {
	details := []string{i18n.T("Version") + " " + p.App().Version}
	if p.UsesSecrets() {
		details = append(details, i18n.T("Store credentials in Keychain"))
	}
	if rules := p.NetworkRules(); rules != "" {
		details = append(details, i18n.T("Network access"), rules)
	}
	install := i18n.T("Install")
	base.ShowAlertDialog(cx, base.AlertDialog{Title: p.App().Name, Description: strings.Join(details, "\n"),
		Actions: []core.AlertAction{{Label: install}, {Label: i18n.T("Cancel"), Role: core.AlertCancel}},
	}, func(choice string) {
		if choice == install {
			m.installNSGoPlugin(store, p)
		}
	})
}

func (m *model) installNSGoPlugin(store *plugin.NSGoStore, p *plugin.NSGoPackage) {
	if m.nsgoPluginBusy.GetUntracked() {
		return
	}
	m.nsgoPluginBusy.Set(true)
	m.cx.Post(func() {
		source, err := plugin.LoadNSGo(m.cx, p)
		if err != nil {
			m.nsgoPluginBusy.Set(false)
			m.report(err)
			return
		}
		go func() {
			err := store.Install(p)
			m.cx.Post(func() {
				m.nsgoPluginBusy.Set(false)
				if err != nil {
					source.Close()
					m.report(err)
					return
				}
				sources := slices.Clone(m.plugins.GetUntracked())
				id := source.Info().ID
				at := slices.IndexFunc(sources, func(s *plugin.NSGoSource) bool { return s.Info().ID == id })
				if at >= 0 {
					sources[at].Close()
					sources[at] = source
				} else {
					sources = append(sources, source)
				}
				m.plugins.Set(sources)
				if m.page.GetUntracked().page != "plugin-market" && m.source.GetUntracked() == id {
					m.refreshAccount(source)
				}
				if dialog := m.signInDialog.GetUntracked(); dialog.source != nil && dialog.source.Info().ID == id && dialog.source != source {
					m.closeSignInDialog()
				}
			})
		}()
	})
}

func (m *model) removeNSGoPlugin(source *plugin.NSGoSource) {
	store := m.nsgoStore.GetUntracked()
	id := source.Info().ID
	if store == nil || m.nsgoPluginBusy.GetUntracked() || m.sourceByID(id) != source {
		return
	}
	m.nsgoPluginBusy.Set(true)
	go func() {
		err := store.Remove(id)
		m.cx.Post(func() {
			m.nsgoPluginBusy.Set(false)
			if err != nil {
				m.report(err)
				return
			}
			if m.source.GetUntracked() == id {
				if m.page.GetUntracked().page == "plugin-market" {
					m.source.Set(local.ID)
				} else {
					m.selectSource(local.ID)
				}
			}
			if m.signInDialog.GetUntracked().source == source {
				m.closeSignInDialog()
			}
			source.Close()
			sources := slices.Clone(m.plugins.GetUntracked())
			m.plugins.Set(slices.DeleteFunc(sources, func(s *plugin.NSGoSource) bool { return s == source }))
		})
	}()
}

func (m *model) nsgoPluginSettings(cx *core.Scope) core.Node {
	install := button(i18n.T("Install NSGo Package"), "plugin-nsgo-install", lucide.FileArchive, func() { m.pickNSGoPlugin(cx) })
	install.Disabled = func() bool { return m.nsgoStore.Get() == nil || m.nsgoPluginBusy.Get() }
	return core.View{Style: core.Style{Gap: 9}, Children: []core.Node{row(install), core.Dynamic(func(cx *core.Scope) core.Node {
		var rows []core.Node
		for _, source := range m.plugins.Get() {
			info := source.Info()
			open := button(i18n.T("Open Source"), "plugin-open-"+info.ID, lucide.ArrowRight, func() { m.selectSource(info.ID) })
			remove := iconButton(lucide.Trash, i18n.T("Remove Plugin"), "plugin-remove-"+info.ID, func() {
				confirm(cx, i18n.T("Remove Plugin?"), info.Name, func() { m.removeNSGoPlugin(source) })
			})
			remove.Disabled = m.nsgoPluginBusy.Get
			rows = append(rows, settingsRow(cx, strings.TrimSpace(info.Name+"  "+info.Version), row(open, remove)))
		}
		return core.Group(rows)
	})}}
}
