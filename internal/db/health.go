package db

import "context"

func (p *Pool) Healthy(ctx context.Context) error { return p.Ping(ctx) }
