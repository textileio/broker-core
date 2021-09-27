package common

import (
	"testing"

	"github.com/filecoin-project/go-address"
	"github.com/stretchr/testify/require"
)

func TestStringifyAddr(t *testing.T) {
	t.Parallel()

	tests := []string{"f0001", "f144zep4gitj73rrujd3jw6iprljicx6vl4wbeavi"}
	for _, test := range tests {
		test := test
		t.Run(test, func(t *testing.T) {
			t.Parallel()

			testAddr, err := address.NewFromString(test)
			require.NoError(t, err)

			str := StringifyAddr(testAddr)
			require.Equal(t, address.MainnetPrefix, str[:1])
		})
	}
}
