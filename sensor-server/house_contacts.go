package sensorserver

type Contact struct {
	ContactAndBatteryLevel uint8  // 0-1 & 0-127
	RSSI                   uint8  // -100 to 0 dBm
	BootTime               uint16 // ms, 0-60000
}

type ContactType uint8

const (
	CONTACT_1     ContactType = 0
	CONTACT_2     ContactType = 1
	CONTACT_3     ContactType = 2
	CONTACT_4     ContactType = 3
	CONTACT_5     ContactType = 4
	CONTACT_6     ContactType = 5
	CONTACT_7     ContactType = 6
	CONTACT_8     ContactType = 7
	CONTACT_9     ContactType = 8
	CONTACT_10    ContactType = 9
	CONTACT_11    ContactType = 10
	CONTACT_12    ContactType = 11
	CONTACT_13    ContactType = 12
	CONTACT_14    ContactType = 13
	CONTACT_15    ContactType = 14
	CONTACT_16    ContactType = 15
	CONTACT_17    ContactType = 16
	CONTACT_18    ContactType = 17
	CONTACT_19    ContactType = 18
	CONTACT_20    ContactType = 19
	CONTACT_21    ContactType = 20
	CONTACT_22    ContactType = 21
	CONTACT_23    ContactType = 22
	CONTACT_24    ContactType = 23
	CONTACT_25    ContactType = 24
	CONTACT_26    ContactType = 25
	CONTACT_27    ContactType = 26
	CONTACT_28    ContactType = 27
	CONTACT_29    ContactType = 28
	CONTACT_30    ContactType = 29
	CONTACT_31    ContactType = 30
	CONTACT_32    ContactType = 31
	CONTACT_COUNT ContactType = 32
)
