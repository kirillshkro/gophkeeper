// Package gophkeeper is the GophKeeper system — a client-server
// architecture for securely storing and synchronizing private user
// information.
//
// The system stores the following data: login/password pairs,
// arbitrary text data, arbitrary binary data and bank card data.
// For any data there is an option to store arbitrary textual
// metadata (belonging to a website, person or bank, one-time
// activation codes, etc.).
//
// The system consists of a server (user registration,
// authentication, authorization, private data storage, data
// synchronization between authorized clients, and transferring
// private data to the owner on request) and a cross-platform
// CLI client (Windows, Linux, Mac OS) for user authentication,
// private data access on request, data management, and displaying
// version and build date of the client binary.
//
// Package gophkeeper is the module root and contains the
// documentation of the whole system.
package gophkeeper
