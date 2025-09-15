// +build windows

// Package jobapi provides supplemental types and functions for low-level
// interactions with the operating system to control job objects.
//
// Golang naming convention is sacrificed in favor of straight name mapping:
// WinAPI uses mixed ALL_CAPS and CaseCamel. To avoid any confusion, naming
// conforms Microsoft documentation.
package jobapi
