// SPDX-License-Identifier: AGPL-3.0-only

package api

import (
	"context"

	"github.com/ranen/dock-weaver/internal/domain"
)

type contextKey string

const userContextKey contextKey = "authenticated-user"

func withUser(ctx context.Context, user domain.User) context.Context {
	return context.WithValue(ctx, userContextKey, user)
}

func userFromContext(ctx context.Context) (domain.User, bool) {
	user, ok := ctx.Value(userContextKey).(domain.User)
	return user, ok
}
