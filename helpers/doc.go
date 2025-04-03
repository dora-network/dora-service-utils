// The matchmaking helpers package contains structures with unexported fields
// designed to be initialized using outside data (orders, prices, etc) then passed
// into matchmaking. By hiding the fields, we allow implementations to change
// without breaking matchmaking code or the packages that import it.
package helpers
