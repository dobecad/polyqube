package availabilityzone

func GetAZsFromRegion(region string) []string {
	switch region {
	case "us-east-1":
		return US_EAST_1_AvailabilityZones
	case "us-east-2":
		return US_EAST_2_AvailabilityZones
	default:
		panic("Unsupported region")
	}
}
