// Package slug converts ticket titles into stable filename slugs.
//
// Slugs are derived from the first significant words of a title, lowercased,
// stripped to ASCII letters and digits, and joined with hyphens. The package is
// deterministic and contains no filesystem or repository state.
package slug
