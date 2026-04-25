package graphql

import (
	"fmt"
	"io"
	"strconv"
	"time"

	"github.com/99designs/gqlgen/graphql"
	"github.com/google/uuid"
)

// DateTime is a type alias for time.Time used by GraphQL.
type DateTime = time.Time

// UUID is a type alias for uuid.UUID used by GraphQL.
type UUID = uuid.UUID

// MarshalDateTime marshals a time.Time to GraphQL DateTime scalar.
func MarshalDateTime(t time.Time) graphql.Marshaler {
	if t.IsZero() {
		return graphql.Null
	}
	return graphql.WriterFunc(func(w io.Writer) {
		io.WriteString(w, strconv.Quote(t.Format(time.RFC3339Nano)))
	})
}

// UnmarshalDateTime unmarshals a GraphQL DateTime scalar to time.Time.
func UnmarshalDateTime(v any) (time.Time, error) {
	if tmpStr, ok := v.(string); ok {
		return time.Parse(time.RFC3339Nano, tmpStr)
	}
	return time.Time{}, fmt.Errorf("invalid datetime type: %T", v)
}

// MarshalUUID marshals a uuid.UUID to GraphQL UUID scalar.
func MarshalUUID(u uuid.UUID) graphql.Marshaler {
	if u == uuid.Nil {
		return graphql.Null
	}
	return graphql.WriterFunc(func(w io.Writer) {
		io.WriteString(w, strconv.Quote(u.String()))
	})
}

// UnmarshalUUID unmarshals a GraphQL UUID scalar to uuid.UUID.
func UnmarshalUUID(v any) (uuid.UUID, error) {
	switch v := v.(type) {
	case string:
		return uuid.Parse(v)
	case []byte:
		return uuid.ParseBytes(v)
	default:
		return uuid.Nil, fmt.Errorf("invalid uuid type: %T", v)
	}
}
