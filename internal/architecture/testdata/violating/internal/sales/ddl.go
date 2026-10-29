// Fixture: lowercase runtime DDL and a grant in business code. Never compiled.
package sales

var (
	alter = "alter table sales.x add column y int"
	grant = "GRANT INSERT ON sales.x TO someone"
)
