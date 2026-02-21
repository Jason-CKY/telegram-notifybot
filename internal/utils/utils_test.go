package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsUsernameAllowed_EmptyWhitelist(t *testing.T) {
	original := WhitelistedUsernames
	WhitelistedUsernames = []string{}
	defer func() { WhitelistedUsernames = original }()

	tests := []struct {
		username string
		expected bool
	}{
		{"anyuser", true},
		{"", true},
		{"random", true},
	}

	for _, tt := range tests {
		result := IsUsernameAllowed(tt.username)
		assert.Equal(t, tt.expected, result, "IsUsernameAllowed(%q) = %v, want %v", tt.username, result, tt.expected)
	}
}

func TestIsUsernameAllowed_WithWhitelist(t *testing.T) {
	original := WhitelistedUsernames
	WhitelistedUsernames = []string{"user1", "user2", "user3"}
	defer func() { WhitelistedUsernames = original }()

	tests := []struct {
		username string
		expected bool
	}{
		{"user1", true},
		{"user2", true},
		{"user3", true},
		{"USER1", true},
		{"User2", true},
		{"user4", false},
		{"", false},
	}

	for _, tt := range tests {
		result := IsUsernameAllowed(tt.username)
		assert.Equal(t, tt.expected, result, "IsUsernameAllowed(%q) = %v, want %v", tt.username, result, tt.expected)
	}
}

func TestLookupEnvStringArray_NotSet(t *testing.T) {
	t.Setenv("TEST_NONEXISTENT_VAR", "")
	result := LookupEnvStringArray("TEST_NONEXISTENT_VAR")
	assert.Empty(t, result)
}

func TestLookupEnvStringArray_Set(t *testing.T) {
	t.Setenv("TEST_ARRAY_VAR", "a,b,c")
	result := LookupEnvStringArray("TEST_ARRAY_VAR")
	require.Len(t, result, 3)
	assert.Equal(t, []string{"a", "b", "c"}, result)
}

func TestIsCurrencySupported(t *testing.T) {
	tests := []struct {
		currency string
		expected bool
	}{
		{"USD", true},
		{"EUR", true},
		{"GBP", true},
		{"JPY", true},
		{"MYR", true},
		{"HKD", true},
		{"AUD", true},
		{"KRW", true},
		{"TWD", true},
		{"IDR", true},
		{"THB", true},
		{"CNY", true},
		{"INR", true},
		{"PHP", true},
		{"usd", true},
		{"eur", true},
		{"Usd", true},
		{"XXX", false},
		{"ABC", false},
		{"", false},
	}

	for _, tt := range tests {
		result := IsCurrencySupported(tt.currency)
		assert.Equal(t, tt.expected, result, "IsCurrencySupported(%q) = %v, want %v", tt.currency, result, tt.expected)
	}
}

func TestSupportedCurrencies_ContainsAll(t *testing.T) {
	expected := []string{"USD", "EUR", "GBP", "JPY", "MYR", "HKD", "AUD", "KRW", "TWD", "IDR", "THB", "CNY", "INR", "PHP"}
	assert.ElementsMatch(t, SupportedCurrencies, expected)
}

func TestHELPMessage_ContainsAllCommands(t *testing.T) {
	commands := []string{
		"/fx",
		"/fx_chart",
		"/fx_subscribe",
		"/fx_interval",
		"/fx_list",
		"/fx_unsubscribe",
	}

	for _, cmd := range commands {
		assert.Contains(t, HELP_MESSAGE, cmd, "HELP_MESSAGE should contain %s", cmd)
	}
}

func TestHELPMessage_ContainsAllCurrencies(t *testing.T) {
	for _, currency := range SupportedCurrencies {
		assert.Contains(t, HELP_MESSAGE, currency, "HELP_MESSAGE should contain %s", currency)
	}
}

func TestHELPMessage_DescribesDataSource(t *testing.T) {
	assert.Contains(t, HELP_MESSAGE, "daily")
}
