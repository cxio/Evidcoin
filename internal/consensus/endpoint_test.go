package consensus

import "testing"

func TestEndpointKinds(t *testing.T) {
	// 协议级与共约级标识值不同
	if EndpointProtocol == EndpointConvention {
		t.Error("EndpointProtocol and EndpointConvention must have different values")
	}
}
