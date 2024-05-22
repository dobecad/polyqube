package utils

type InstanceType int

// Define compatable EC2 instance types suitable for hosting large, HA K3S clusters
// At a minimum, the type must have 2 vCPUs and 4GB RAM
const (
	T2_large InstanceType = iota
	T2_xlarge
	T2_2xlarge

	T3_large
	T3_xlarge
	T3_2xlarge

	T3a_large
	T3a_xlarge
	T3a_2xlarge

	M4_large
	M4_xlarge
	M4_2xlarge
	M4_4xlarge
	M4_10xlarge

	M5a_large
	M5a_xlarge
	M5a_2xlarge
	M5a_4xlarge
	M5a_8xlarge
)

func (i InstanceType) String() string {
	switch i {
	case T2_large:
		return "t2.large"
	case T2_xlarge:
		return "t2.xlarge"
	case T2_2xlarge:
		return "t2.2xlarge"
	case T3_large:
		return "t3.large"
	case T3_xlarge:
		return "t3.xlarge"
	case T3_2xlarge:
		return "t3.2xlarge"
	case T3a_large:
		return "t3a.large"
	case T3a_xlarge:
		return "t3a.xlarge"
	case T3a_2xlarge:
		return "t3a.2xlarge"
	case M4_large:
		return "m4.large"
	case M4_xlarge:
		return "m4.xlarge"
	case M4_2xlarge:
		return "m4.2xlarge"
	case M4_4xlarge:
		return "m4.4xlarge"
	case M4_10xlarge:
		return "m4.10xlarge"
	case M5a_large:
		return "m5a.large"
	case M5a_xlarge:
		return "m5a.xlarge"
	case M5a_2xlarge:
		return "m5a.2xlarge"
	case M5a_4xlarge:
		return "M5a.4xlarge"
	case M5a_8xlarge:
		return "m5a.8xlarge"
	default:
		panic("unknown variant")
	}
}
