package password

import (
	"crypto/rand" // For secure random number generation (essential for passwords)
	"fmt"
	"math/big"
	math_rand "math/rand" // For shuffling the password (order doesn't need cryptographic randomness)
	"time"                // Used to seed math/rand for shuffling
)

// PasswordConfig holds the configuration parameters for generating a password.
type PasswordConfig struct {
	maximumLength int // Enforce a maximum length (if not set to 0)
	minimumLength int // Enforce a minimum length (if not set to 0)

	length       int // Total length of the password
	numSymbols   int // Minimum number of symbols
	numDigits    int // Minimum number of digits
	numUppercase int // Minimum number of uppercase characters
	numLowercase int // Minimum number of lowercase characters

	// Symbols to use within password generation
	symbols string
}

// PasswordOption is a function type that takes a pointer to a PasswordConfig
// and modifies it. This is the core of the function options pattern.
type PasswordOption func(*PasswordConfig) error

// WithLength sets the total length of the password.
// It ensures that the provided length is positive.
func WithLength(length int) PasswordOption {
	return func(c *PasswordConfig) error {
		if length < 1 {
			return fmt.Errorf("password length must be positive")
		}
		c.length = length
		return nil
	}
}

// WithMaximumLengthConstraint enforces a maximum password length
func WithMaximumLengthConstraint(length int) PasswordOption {
	return func(c *PasswordConfig) error {
		if length > 0 {
			c.maximumLength = length
		}
		return nil
	}
}

// WithMinimumLengthConstraint enforces a minimum password length
func WithMinimumLengthConstraint(length int) PasswordOption {
	return func(c *PasswordConfig) error {
		if length > 0 {
			c.minimumLength = length
		}
		return nil
	}
}

// WithLengthConstraints enforces a minimum and maximum password length
func WithLengthConstraints(minimum int, maximum int) PasswordOption {
	return func(c *PasswordConfig) error {
		// Don't allow setting the min below 1
		c.minimumLength = max(1, minimum)
		c.maximumLength = maximum
		return nil
	}
}

// WithUrlCompatibleSymbols sets the number of symbols to use when generating a password and use symbols that
// don't require additional URL encoding
func WithUrlCompatibleSymbols(num int) PasswordOption {
	return func(c *PasswordConfig) error {
		if num >= 0 {
			c.numSymbols = num
			c.symbols = symbolUrlWithoutEncodingChars
		}
		return nil
	}
}

// WithShellCompatibleSymbols sets the number of symbols to use when generating a password and use symbols that
// don't require additional shell escaping
func WithShellCompatibleSymbols(num int) PasswordOption {
	return func(c *PasswordConfig) error {
		if num >= 0 {
			c.numSymbols = num
			c.symbols = symbolShellCompatibleChars
		}
		return nil
	}
}

// WithCustomSymbols sets the number of symbols to use when generating a password and use
// the given set of symbols
func WithCustomSymbols(symbols string, num int) PasswordOption {
	return func(c *PasswordConfig) error {
		if num >= 0 {
			c.numSymbols = num
			c.symbols = symbols
		}
		return nil
	}
}

// WithSymbols sets the minimum number of symbols to include in the password.
// It ensures the number of symbols is non-negative.
func WithSymbols(num int) PasswordOption {
	return func(c *PasswordConfig) error {
		if num >= 0 {
			c.numSymbols = num
		}
		return nil
	}
}

// WithDigits sets the minimum number of digits to include in the password.
// It ensures the number of digits is non-negative.
func WithDigits(num int) PasswordOption {
	return func(c *PasswordConfig) error {
		if num >= 0 {
			c.numDigits = num
		}
		return nil
	}
}

// WithUppercase sets the minimum number of uppercase characters to include.
// It ensures the number of uppercase characters is non-negative.
func WithUppercase(num int) PasswordOption {
	return func(c *PasswordConfig) error {
		if num >= 0 {
			c.numUppercase = num
		}
		return nil
	}
}

// WithLowercase sets the minimum number of lowercase characters to include.
// It ensures the number of lowercase characters is non-negative.
func WithLowercase(num int) PasswordOption {
	return func(c *PasswordConfig) error {
		if num >= 0 {
			c.numLowercase = num
		}
		return nil
	}
}

// Define character sets for different types of characters.
const (
	lowercaseChars = "abcdefghijklmnopqrstuvwxyz"
	uppercaseChars = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	digitChars     = "0123456789"
	symbolChars    = "`!$@#%^&*()-_=+[]{}|;:,.<>/?~"

	// Symbols which don't require url encoding
	symbolUrlWithoutEncodingChars = "-._~"

	// shell compatible symbols (which don't require escaping with either double or single quotes)
	symbolShellCompatibleChars = "@#%^&*()-_=+[]{}|;:,.<>/?~"
)

// getRandomChar securely picks a random character from the given charset.
// It uses crypto/rand for high-quality randomness, suitable for security-sensitive operations.
func getRandomChar(charset string) (byte, error) {
	if len(charset) == 0 {
		return 0, fmt.Errorf("charset is empty, cannot pick a character")
	}
	// Generate a cryptographically secure random index within the charset length.
	idx, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
	if err != nil {
		return 0, fmt.Errorf("failed to generate random index for character selection: %w", err)
	}
	return charset[idx.Int64()], nil
}

// shuffleBytes shuffles a slice of bytes randomly.
// It uses math/rand, which is sufficient for shuffling as the character selection
// has already been done securely.
func shuffleBytes(b []byte) {
	// Seed math/rand with the current time to ensure different shuffles each run.
	r := math_rand.New(math_rand.NewSource(time.Now().UnixNano()))
	r.Shuffle(len(b), func(i, j int) {
		b[i], b[j] = b[j], b[i]
	})
}

// NewRandomPassword generates a new random password based on the provided options.
// It returns the generated password string and an error if any validation or
// generation step fails.
func NewRandomPassword(opts ...PasswordOption) (string, error) {
	// Initialize a default password configuration.
	// These defaults are applied if no specific options are provided by the user.
	config := PasswordConfig{
		maximumLength: 0, // Enforce a maximum length when not set to 0
		minimumLength: 1, // Enforce a minimum length when not set to 0

		length:       31, // Default password length
		numSymbols:   2,  // Default minimum symbols
		numDigits:    2,  // Default minimum digits
		numUppercase: 2,  // Default minimum uppercase characters
		numLowercase: 2,  // Default minimum lowercase characters
		symbols:      symbolChars,
	}

	// Apply any user-provided options. These will override the default values.
	for _, opt := range opts {
		if err := opt(&config); err != nil {
			return "", err
		}
	}

	// Password length enforcement
	if config.maximumLength > 0 {
		if config.length > config.maximumLength {
			return "", fmt.Errorf("password exceeds the maximum allowed length. len=%d, max=%d", config.length, config.maximumLength)
		}
	}
	if config.length < config.minimumLength {
		return "", fmt.Errorf("password must length is not sufficient. len=%d, min=%d", config.length, config.minimumLength)
	}

	// Validate the final configuration.
	requiredChars := config.numSymbols + config.numDigits + config.numUppercase + config.numLowercase
	if requiredChars > config.length {
		return "", fmt.Errorf("sum of required characters (%d) exceeds total password length (%d)", requiredChars, config.length)
	}

	if requiredChars <= 0 {
		return "", fmt.Errorf("sum of character types must be positive")
	}

	if config.length <= 0 {
		return "", fmt.Errorf("password length must be positive")
	}

	// Create a byte slice to build the password.
	password := make([]byte, config.length)
	currentIdx := 0 // Tracks the current position in the password slice

	// Add the required number of symbols.
	for i := 0; i < config.numSymbols; i++ {
		char, err := getRandomChar(config.symbols)
		if err != nil {
			return "", fmt.Errorf("failed to add required symbol: %w", err)
		}
		password[currentIdx] = char
		currentIdx++
	}

	// Add the required number of digits.
	for i := 0; i < config.numDigits; i++ {
		char, err := getRandomChar(digitChars)
		if err != nil {
			return "", fmt.Errorf("failed to add required digit: %w", err)
		}
		password[currentIdx] = char
		currentIdx++
	}

	// Add the required number of uppercase characters.
	for i := 0; i < config.numUppercase; i++ {
		char, err := getRandomChar(uppercaseChars)
		if err != nil {
			return "", fmt.Errorf("failed to add required uppercase character: %w", err)
		}
		password[currentIdx] = char
		currentIdx++
	}

	// Add the required number of lowercase characters.
	for i := 0; i < config.numLowercase; i++ {
		char, err := getRandomChar(lowercaseChars)
		if err != nil {
			return "", fmt.Errorf("failed to add required lowercase character: %w", err)
		}
		password[currentIdx] = char
		currentIdx++
	}

	// Build a pool of all allowed characters for filling the remaining length.
	// This pool includes character types that have a non-zero minimum requirement,
	// or all types if no specific requirements were set (to ensure a varied password).
	allCharsPool := ""
	if config.numLowercase > 0 {
		allCharsPool += lowercaseChars
	}
	if config.numUppercase > 0 {
		allCharsPool += uppercaseChars
	}
	if config.numDigits > 0 {
		allCharsPool += digitChars
	}
	if config.numSymbols > 0 {
		allCharsPool += config.symbols
	}

	// If no specific character types were requested, and the password length is positive,
	// default to using all character types for the remaining length to ensure variety.
	if allCharsPool == "" && config.length > 0 {
		allCharsPool = lowercaseChars + uppercaseChars + digitChars + config.symbols
	}

	// If after all logic, the character pool is still empty but length is positive,
	// it means no characters can be generated.
	if allCharsPool == "" && config.length > 0 {
		return "", fmt.Errorf("no character types enabled for password generation, cannot fill remaining length")
	}

	// Fill the remaining length of the password with random characters from the combined pool.
	for i := currentIdx; i < config.length; i++ {
		char, err := getRandomChar(allCharsPool)
		if err != nil {
			return "", fmt.Errorf("failed to fill remaining password length: %w", err)
		}
		password[i] = char
	}

	// Shuffle the entire password byte slice to randomize the positions of characters.
	// This ensures that the required characters are not always at the beginning.
	shuffleBytes(password)

	// Convert the byte slice to a string and return.
	return string(password), nil
}
