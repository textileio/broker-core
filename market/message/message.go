package message

import "github.com/oklog/ulid/v2"

type Type string

type MarketMessage interface {
	ID() ulid.ULID
	Type() Type
}
