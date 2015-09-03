package fs

import (
	"github.com/omeid/gonzo/context"
	"github.com/omeid/kargar"
)

func Copy(dst string, sources ...string) kargar.Action {
	return func(ctx context.Context) error {
		return Src(ctx, sources...).Then(Dest(dst))
	}
}
