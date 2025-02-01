// Package versions
package versions

var (
	// Version holds the EmbeddedCluster version.
	Version = "v0.0.0"
	// K0sVersion holds the version of k0s binary we are embedding. this is
	// set at compile time via ldflags.
	K0sVersion = "0.0.0"
	// TroubleshootVersion holds the version of troubleshoot and preflight
	// binaries we are embedding. this is set at compile time via ldflags.
	TroubleshootVersion = "0.0.0"
	// LocalArtifactMirrorImage holds a reference to where the lam image for
	// this version of embedded-cluster is stored. Set at compile time.
	LocalArtifactMirrorImage = ""

	// K0sBinaryURLOverride is used to override the k0s binary url and is set at compile time using
	// LD_FLAGS in the Makefile
	K0sBinaryURLOverride string
	// KOTSBinaryURLOverride is used to override the KOTS binary url and is set at compile time
	// using LD_FLAGS in the Makefile
	KOTSBinaryURLOverride string
	// OperatorBinaryURLOverride is used to override the Operator binary url and is set at compile
	// time using LD_FLAGS in the Makefile
	OperatorBinaryURLOverride string
)
