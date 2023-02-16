package client

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFakeClient(t *testing.T) {
	expectedResult := []Release{
		{
			TagName: "a",
		},
	}
	expectedErr := fmt.Errorf("errr")
	result, err := NewFakeClient(expectedResult, expectedErr).Releases("s")
	require.Equal(t, expectedErr, err)
	require.Equal(t, expectedResult, result)
}
