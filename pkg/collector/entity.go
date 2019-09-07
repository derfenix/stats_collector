package collector

// Entity statistics entity will be written into database
type Entity struct {
	ID    uint64 `sql:"id"`
	Label string `sql:"label"`
	Count uint64 `sql:"count"`

	// attempt is entity's amount of write attempts
	attempt uint8
}
