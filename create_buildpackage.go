package pack

import (
	"context"
	"io"

	"github.com/buildpack/imgutil"
	"github.com/pkg/errors"

	"github.com/buildpack/pack/internal/buildpackage"
	"github.com/buildpack/pack/internal/dist"
	"github.com/buildpack/pack/internal/style"
)

type CreatePackageOptions struct {
	Name    string
	Config  buildpackage.Config
	Publish bool
	NoPull  bool
}

func (c *Client) CreatePackage(ctx context.Context, opts CreatePackageOptions) error {
	packageBuilder := buildpackage.NewBuilder(c.imageFactory)

	for _, bc := range opts.Config.Buildpacks {
		blob, err := c.downloader.Download(ctx, bc.URI)
		if err != nil {
			return errors.Wrapf(err, "downloading buildpack from %s", style.Symbol(bc.URI))
		}

		bp, err := dist.NewBuildpack(blob)
		if err != nil {
			return errors.Wrapf(err, "creating buildpack from %s", style.Symbol(bc.URI))
		}

		packageBuilder.AddBuildpack(bp)
	}

	for _, ref := range opts.Config.Packages {
		pkgImage, err := c.imageFetcher.Fetch(ctx, ref.Ref, !opts.Publish, !opts.NoPull)
		if err != nil {
			return errors.Wrapf(err, "fetching image %s", style.Symbol(ref.Ref))
		}

		bpLayers := dist.BuildpackLayers{}
		ok, err := dist.GetLabel(pkgImage, dist.BuildpackLayersLabel, &bpLayers)
		if err != nil {
			return err
		}

		if !ok {
			return errors.Errorf(
				"label %s not present on package %s",
				style.Symbol(dist.BuildpackLayersLabel),
				style.Symbol(ref.Ref),
			)
		}

		pkg := &imgUtilPackage{
			img:      pkgImage,
			bpLayers: bpLayers,
		}

		packageBuilder.AddPackage(pkg)
	}

	packageBuilder.SetDefaultBuildpack(opts.Config.Default)

	for _, s := range opts.Config.Stacks {
		packageBuilder.AddStack(s)
	}

	_, err := packageBuilder.Save(opts.Name, opts.Publish)
	if err != nil {
		return errors.Wrapf(err, "saving image")
	}

	return err
}

type imgUtilPackage struct {
	img      imgutil.Image
	bpLayers dist.BuildpackLayers
}

func (i *imgUtilPackage) Name() string {
	return i.img.Name()
}

func (i *imgUtilPackage) BuildpackLayers() dist.BuildpackLayers {
	return i.bpLayers
}

func (i *imgUtilPackage) GetLayer(diffID string) (io.ReadCloser, error) {
	return i.img.GetLayer(diffID)
}

func (i *imgUtilPackage) Label(name string) (value string, err error) {
	return i.img.Label(name)
}
