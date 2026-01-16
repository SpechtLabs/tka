//go:build archgo
// +build archgo

// Package main defines architectural rules using arch-go.
// Run with: arch-go
//
// Install: go install github.com/fdaines/arch-go@latest
//
// This file enforces clean architecture boundaries and dependency rules.
package main

import (
	"github.com/fdaines/arch-go/api"
	"github.com/fdaines/arch-go/api/configuration"
)

// ArchitectureRules defines all architectural constraints for TKA
func ArchitectureRules() *configuration.Config {
	return &configuration.Config{
		Version: 1,
		// Threshold for coverage requirements
		Threshold: &configuration.Threshold{
			Compliance: &configuration.ThresholdRule{
				Rate:  80,
				Scope: "package",
			},
			Coverage: &configuration.ThresholdRule{
				Rate:  80,
				Scope: "package",
			},
		},

		// Define dependency rules
		DependenciesRules: []*configuration.DependenciesRule{
			// Operators should not import net/http directly
			{
				Package:              "github.com/spechtlabs/tka/pkg/operator",
				ShouldNotDependsOn:   []string{"net/http", "database/sql"},
				ShouldOnlyDependsOn:  nil,
				ShouldNotDependsOnExternal: []string{
					"github.com/gin-gonic/gin", // Operators shouldn't know about HTTP framework
				},
			},

			// API layer can use gin but not Kubernetes internals directly
			{
				Package: "github.com/spechtlabs/tka/pkg/service/api",
				ShouldNotDependsOn: []string{
					"k8s.io/client-go/kubernetes",  // Use client abstraction
					"sigs.k8s.io/controller-runtime", // This is for operators only
				},
			},

			// Client package should not import service layer (prevents circular deps)
			{
				Package: "github.com/spechtlabs/tka/pkg/client/**",
				ShouldNotDependsOn: []string{
					"github.com/spechtlabs/tka/pkg/service/**",
					"github.com/spechtlabs/tka/pkg/operator/**",
				},
			},

			// Models should be pure data, no external dependencies
			{
				Package: "github.com/spechtlabs/tka/pkg/models",
				ShouldOnlyDependsOn: []string{
					"github.com/sierrasoftworks/humane-errors-go", // For error handling
					"encoding/json", // For serialization
				},
			},

			// Service models should only depend on standard library
			{
				Package: "github.com/spechtlabs/tka/pkg/service/models",
				ShouldNotDependsOn: []string{
					"k8s.io/**",
					"sigs.k8s.io/**",
				},
			},

			// Internal packages should not be imported by external cmd packages
			{
				Package: "github.com/spechtlabs/tka/internal/**",
				ShouldNotDependsOn: []string{
					"github.com/spechtlabs/tka/cmd/**",
				},
			},

			// Middleware should be self-contained
			{
				Package: "github.com/spechtlabs/tka/pkg/middleware/**",
				ShouldNotDependsOn: []string{
					"github.com/spechtlabs/tka/pkg/operator/**",
					"github.com/spechtlabs/tka/pkg/client/**",
				},
			},

			// tshttp package should be independent of business logic
			{
				Package: "github.com/spechtlabs/tka/pkg/tshttp",
				ShouldNotDependsOn: []string{
					"github.com/spechtlabs/tka/pkg/service/**",
					"github.com/spechtlabs/tka/pkg/operator/**",
					"github.com/spechtlabs/tka/pkg/client/**",
				},
			},
		},

		// Content rules - naming conventions and patterns
		ContentRules: []*configuration.ContentsRule{
			// Interfaces should be in interface.go or *_interface.go files
			{
				Package:                 "github.com/spechtlabs/tka/pkg/**",
				ShouldOnlyContainInterfaces: true,
				InFiles:                 []string{"interface.go", "interfaces.go", "*_interface.go"},
			},

			// Mock implementations should be in mock/ subdirectories
			{
				Package:              "github.com/spechtlabs/tka/**/mock",
				ShouldOnlyContainStructs: true,
			},
		},

		// Function rules
		FunctionRules: []*configuration.FunctionsRule{
			// Exported functions should be limited in line count
			{
				Package: "github.com/spechtlabs/tka/pkg/**",
				MaxLines: 50, // Keep functions small and focused
			},

			// Public API functions should be well-documented
			{
				Package:             "github.com/spechtlabs/tka/pkg/service/api",
				MaxPublicFunctions:  20, // Limit API surface
			},
		},

		// Naming rules
		NamingRules: []*configuration.NamingRule{
			// Interfaces should have descriptive names
			{
				Package: "github.com/spechtlabs/tka/pkg/**",
				InterfaceImplementationNamingRule: &configuration.InterfaceImplementationNamingRule{
					ShouldHaveSimpleNameEndingWith: []string{"Impl", "Service", "Client", "Handler"},
				},
			},
		},
	}
}
