package providers

import "github.com/vaayne/mcphub/internal/skills"

func init() {
	// Register providers in priority order
	// Mintlify first (more specific - checks for mintlify-proj)
	// Then HuggingFace (domain-specific)
	// Then Direct (fallback for generic URLs)
	skills.RegisterProvider(NewMintlifyProvider())
	skills.RegisterProvider(NewHuggingFaceProvider())
	skills.RegisterProvider(NewDirectProvider())
}
