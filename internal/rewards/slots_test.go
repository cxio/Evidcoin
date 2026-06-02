package rewards

import "testing"

func TestSlots(t *testing.T) {
	t.Run("zero_initial", func(t *testing.T) {
		var s AwardSlots
		if !s.IsZero() {
			t.Error("new AwardSlots should be all-zero")
		}
	})

	t.Run("SetBit_IsSet_basic", func(t *testing.T) {
		var s AwardSlots
		// 设置 bit0（H-1）并验证。
		s.SetBit(ServiceBlockqs, 1)
		if !s.IsSet(ServiceBlockqs, 1) {
			t.Error("bit offset=1 (H-1) should be set for Blockqs")
		}
		// 其他服务的 bit0 未被置位。
		if s.IsSet(ServiceDepots, 1) {
			t.Error("Depots bit1 should not be set")
		}
		if s.IsSet(ServiceSTUN, 1) {
			t.Error("STUN bit1 should not be set")
		}
	})

	t.Run("SetBit_bit47_H-48", func(t *testing.T) {
		var s AwardSlots
		// 设置 bit47（H-48）——每个服务槽的最后一位。
		s.SetBit(ServiceBlockqs, 48)
		if !s.IsSet(ServiceBlockqs, 48) {
			t.Error("Blockqs: bit offset=48 (H-48) should be set")
		}
		// 验证 bit47 对应 Blockqs 槽第 6 字节的 MSB（byte[5] bit7）。
		if s[5]>>7 != 1 {
			t.Errorf("Blockqs byte[5] MSB should be 1, got byte=%08b", s[5])
		}
	})

	t.Run("bit0_maps_to_byte0_bit0", func(t *testing.T) {
		// bit0（offset=1）对应字节 0 的 bit0（LSB），即 byte & 0x01 = 1。
		var s AwardSlots
		s.SetBit(ServiceBlockqs, 1)
		if s[0]&0x01 != 1 {
			t.Errorf("Blockqs byte[0] LSB should be 1, got %08b", s[0])
		}
	})

	t.Run("service_byte_layout", func(t *testing.T) {
		// 验证三个服务各占 6 字节，互不干扰。
		var s AwardSlots
		s.SetBit(ServiceBlockqs, 1) // byte[0] LSB
		s.SetBit(ServiceDepots, 1)  // byte[6] LSB
		s.SetBit(ServiceSTUN, 1)    // byte[12] LSB

		if s[0] != 1 {
			t.Errorf("Blockqs byte[0]=%08b, want 00000001", s[0])
		}
		if s[6] != 1 {
			t.Errorf("Depots byte[6]=%08b, want 00000001", s[6])
		}
		if s[12] != 1 {
			t.Errorf("STUN byte[12]=%08b, want 00000001", s[12])
		}
	})

	t.Run("SetBit_all_offsets", func(t *testing.T) {
		var s AwardSlots
		for offset := 1; offset <= 48; offset++ {
			s.SetBit(ServiceDepots, offset)
		}
		// 所有 bit 置位后，Depots 的 6 字节应全为 0xFF。
		for i := 6; i < 12; i++ {
			if s[i] != 0xFF {
				t.Errorf("Depots byte[%d] = %08b, want 11111111", i, s[i])
			}
		}
		// Blockqs 与 STUN 不受影响。
		for i := 0; i < 6; i++ {
			if s[i] != 0 {
				t.Errorf("Blockqs byte[%d] should be 0, got %08b", i, s[i])
			}
		}
		for i := 12; i < 18; i++ {
			if s[i] != 0 {
				t.Errorf("STUN byte[%d] should be 0, got %08b", i, s[i])
			}
		}
	})

	t.Run("CountForService", func(t *testing.T) {
		var s AwardSlots
		if s.CountForService(ServiceBlockqs) != 0 {
			t.Error("empty slot: count should be 0")
		}
		s.SetBit(ServiceBlockqs, 1)
		s.SetBit(ServiceBlockqs, 3)
		s.SetBit(ServiceBlockqs, 48)
		if got := s.CountForService(ServiceBlockqs); got != 3 {
			t.Errorf("CountForService(Blockqs) = %d, want 3", got)
		}
	})

	t.Run("out_of_range_ignored", func(t *testing.T) {
		var s AwardSlots
		s.SetBit(ServiceBlockqs, 0)  // 超出范围，忽略
		s.SetBit(ServiceBlockqs, 49) // 超出范围，忽略
		if !s.IsZero() {
			t.Error("out-of-range SetBit should be no-op, slot should remain zero")
		}
		if s.IsSet(ServiceBlockqs, 0) {
			t.Error("IsSet(0) should return false (out of range)")
		}
		if s.IsSet(ServiceBlockqs, 49) {
			t.Error("IsSet(49) should return false (out of range)")
		}
	})

	t.Run("IsZero_after_set", func(t *testing.T) {
		var s AwardSlots
		s.SetBit(ServiceSTUN, 24)
		if s.IsZero() {
			t.Error("IsZero should be false after SetBit")
		}
	})
}
