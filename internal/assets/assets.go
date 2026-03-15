package assets

import (
	_ "embed"
)

//go:embed stealth.js
var StealthScript string

//go:embed readability.js
var ReadabilityJS string

//go:embed welcome.html
var WelcomeHTML string

//go:embed viewer.html
var ViewerHTML string
