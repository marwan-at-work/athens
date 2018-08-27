package stasher

import (
	"context"
	"io"
	"time"

	"github.com/gomods/athens/pkg/download"
	"github.com/gomods/athens/pkg/errors"
	"github.com/gomods/athens/pkg/stash"
	"github.com/gomods/athens/pkg/storage"
)

type protocol struct {
	s       storage.Backend
	dp      download.Protocol
	stasher stash.Stasher
}

// New takes an upstream Protocol and storage
// it always prefers storage, otherwise it goes to upstream
// and stashes the storage with the results through the given stasher.
func New(dp download.Protocol, s storage.Backend, stasher stash.Stasher) download.Protocol {
	p := &protocol{dp: dp, s: s, stasher: stasher}

	return p
}

func (p *protocol) List(ctx context.Context, mod string) ([]string, error) {
	return p.dp.List(ctx, mod)
}

func (p *protocol) Info(ctx context.Context, mod, ver string) ([]byte, error) {
	const op errors.Op = "stasher.Info"
	info, err := p.s.Info(ctx, mod, ver)
	if errors.IsNotFoundErr(err) {
		err = p.stasher.Stash(mod, ver)
		if err != nil {
			return nil, errors.E(op, err)
		}
		info, err = p.s.Info(ctx, mod, ver)
	}
	if err != nil {
		return nil, errors.E(op, err)
	}

	return info, nil
}

func (p *protocol) Stash(mod, ver string) error {
	const op errors.Op = "stasher.Stash"
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute*10)
	defer cancel()
	v, err := p.dp.Version(ctx, mod, ver)
	if err != nil {
		return errors.E(op, err)
	}
	defer v.Zip.Close()
	err = p.s.Save(ctx, mod, ver, v.Mod, v.Zip, v.Info)
	if err != nil {
		return errors.E(op, err)
	}
	return nil
}

func (p *protocol) Latest(ctx context.Context, mod string) (*storage.RevInfo, error) {
	const op errors.Op = "stasher.Latest"
	info, err := p.dp.Latest(ctx, mod)
	if err != nil {
		return nil, errors.E(op, err)
	}

	return info, nil
}

func (p *protocol) GoMod(ctx context.Context, mod, ver string) ([]byte, error) {
	const op errors.Op = "stasher.GoMod"
	goMod, err := p.s.GoMod(ctx, mod, ver)
	if errors.IsNotFoundErr(err) {
		err = p.stasher.Stash(mod, ver)
		if err != nil {
			return nil, errors.E(op, err)
		}
		goMod, err = p.s.GoMod(ctx, mod, ver)
	}
	if err != nil {
		return nil, errors.E(op, err)
	}

	return goMod, nil
}

func (p *protocol) Zip(ctx context.Context, mod, ver string) (io.ReadCloser, error) {
	const op errors.Op = "stasher.Zip"
	zip, err := p.s.Zip(ctx, mod, ver)
	if errors.IsNotFoundErr(err) {
		err = p.stasher.Stash(mod, ver)
		if err != nil {
			return nil, errors.E(op, err)
		}
		zip, err = p.s.Zip(ctx, mod, ver)
	}
	if err != nil {
		return nil, errors.E(op, err)
	}

	return zip, nil
}

func (p *protocol) Version(ctx context.Context, mod, ver string) (*storage.Version, error) {
	const op errors.Op = "stasher.Version"
	info, err := p.Info(ctx, mod, ver)
	if err != nil {
		return nil, errors.E(op, err)
	}

	goMod, err := p.GoMod(ctx, mod, ver)
	if err != nil {
		return nil, errors.E(op, err)
	}

	zip, err := p.s.Zip(ctx, mod, ver)
	if err != nil {
		return nil, errors.E(op, err)
	}

	return &storage.Version{
		Info: info,
		Mod:  goMod,
		Zip:  zip,
	}, nil
}