package plugin

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/nsgo-dev/nsgo/tsx/pkg"
)

const nsgoPackageExtension = ".nsgopkg"
const maxNSGoPackage = 8 << 20

var pluginName = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.-]*$`)

// NSGoStore verifies and stores signed music plugins in the application data directory.
type NSGoStore struct {
	dir   string
	trust *pkg.Trust
}

type NSGoPackage struct {
	verified *pkg.Package
	data     []byte
}

func (p *NSGoPackage) App() pkg.App { return p.verified.App }

func (p *NSGoPackage) UsesSecrets() bool { return p.verified.Capabilities.Secrets }

func (p *NSGoPackage) NetworkRules() string {
	if p.verified.Capabilities.Network == nil {
		return ""
	}
	data, _ := json.MarshalIndent(p.verified.Capabilities.Network, "", "  ")
	return string(data)
}

func OpenNSGoStore(dataDir string) (*NSGoStore, error) {
	dir := filepath.Join(dataDir, "nsgo-plugins")
	trust, err := pkg.OpenTrust(dir)
	if err != nil {
		return nil, err
	}
	root, err := pkg.DecodePublicKey(demoRoot)
	if err != nil {
		return nil, err
	}
	trust.AddRoot(root)
	roots, err := filepath.Glob(filepath.Join(dir, "roots", "*.pub"))
	if err != nil {
		return nil, err
	}
	for _, path := range roots {
		data, err := readBounded(path, 1024)
		if err != nil {
			return nil, err
		}
		key, err := pkg.DecodePublicKey(string(data))
		if err != nil {
			return nil, fmt.Errorf("%s: %w", path, err)
		}
		trust.AddRoot(key)
	}
	return &NSGoStore{dir: dir, trust: trust}, nil
}

func (s *NSGoStore) Read(path string) (*NSGoPackage, error) {
	data, err := readBounded(path, maxNSGoPackage)
	if err != nil {
		return nil, err
	}
	return s.readPackage(data)
}

func (s *NSGoStore) readPackage(data []byte) (*NSGoPackage, error) {
	p, err := s.verify(data)
	if err != nil {
		return nil, err
	}
	if !pluginName.MatchString(p.App.ID) {
		return nil, fmt.Errorf("invalid NSGo plugin ID %q", p.App.ID)
	}
	return &NSGoPackage{verified: p, data: data}, nil
}

func (s *NSGoStore) Discover() ([]*NSGoPackage, error) {
	entries, err := os.ReadDir(s.dir)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var packages []*NSGoPackage
	var failures []error
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, nsgoPackageExtension) {
			continue
		}
		p, err := s.Read(filepath.Join(s.dir, name))
		if err == nil && name != p.App().ID+nsgoPackageExtension {
			err = errors.New("filename does not match signed plugin ID")
		}
		if err != nil {
			failures = append(failures, fmt.Errorf("%s: %w", name, err))
			continue
		}
		packages = append(packages, p)
	}
	return packages, errors.Join(failures...)
}

// ReadInstall checks an incoming file and its relationship to an existing package
// before the candidate's code is run.
func (s *NSGoStore) ReadInstall(path string) (*NSGoPackage, error) {
	data, err := readBounded(path, maxNSGoPackage)
	if err != nil {
		return nil, err
	}
	return s.prepareInstall(data)
}

func (s *NSGoStore) prepareInstall(data []byte) (*NSGoPackage, error) {
	p, err := s.readPackage(data)
	if err != nil {
		return nil, err
	}
	_, err = s.replacement(p, p.verified)
	return p, err
}

func (s *NSGoStore) replacement(p *NSGoPackage, candidate *pkg.Package) ([]byte, error) {
	path := filepath.Join(s.dir, p.App().ID+nsgoPackageExtension)
	previous, err := readBounded(path, maxNSGoPackage)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	if previous != nil && !bytes.Equal(previous, p.data) {
		old, err := s.verify(previous)
		if err != nil {
			return nil, fmt.Errorf("verify installed package before replacing it: %w", err)
		}
		oldSigner, err := old.Certificate.Info()
		if err != nil {
			return nil, err
		}
		newSigner, err := candidate.Certificate.Info()
		if err != nil {
			return nil, err
		}
		if old.App.ID != candidate.App.ID || oldSigner.Signer != newSigner.Signer || candidate.App.Build <= old.App.Build {
			return nil, errors.New("a plugin update needs the same signing root and a newer build")
		}
	}
	return previous, nil
}

// Install records a package that has successfully loaded. The caller serializes
// changes and keeps the old source active until this method succeeds.
func (s *NSGoStore) Install(p *NSGoPackage) error {
	candidate, err := s.verify(p.data)
	if err != nil {
		return err
	}
	previous, err := s.replacement(p, candidate)
	if err != nil {
		return err
	}
	path := filepath.Join(s.dir, p.App().ID+nsgoPackageExtension)
	if err := writeNSGoPackage(path, p.data); err != nil {
		return err
	}
	if err := s.trust.Accept(candidate); err != nil {
		if previous == nil {
			return errors.Join(err, os.Remove(path))
		}
		return errors.Join(err, writeNSGoPackage(path, previous))
	}
	return nil
}

func (s *NSGoStore) Remove(id string) error {
	id, ok := strings.CutPrefix(id, NSGoSourcePrefix)
	if !ok || !pluginName.MatchString(id) {
		return errors.New("invalid NSGo source ID")
	}
	return os.Remove(filepath.Join(s.dir, id+nsgoPackageExtension))
}

func readBounded(path string, limit int64) ([]byte, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() || info.Size() > limit {
		return nil, fmt.Errorf("invalid file or exceeds %d bytes: %s", limit, filepath.Base(path))
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, limit+1))
	if int64(len(data)) > limit {
		return nil, fmt.Errorf("file exceeds %d bytes: %s", limit, filepath.Base(path))
	}
	return data, err
}

func writeNSGoPackage(path string, data []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	f, err := os.CreateTemp(dir, ".installing-*.tmp")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	_, writeErr := f.Write(data)
	if err := errors.Join(writeErr, f.Close()); err != nil {
		return err
	}
	return os.Rename(f.Name(), path)
}
