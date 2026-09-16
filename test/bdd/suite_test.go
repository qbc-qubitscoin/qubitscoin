package bdd_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// TestBDDSuite is the entry point for all Ginkgo BDD specs in this package.
func TestBDDSuite(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "QubitsCoin BDD Suite")
}
