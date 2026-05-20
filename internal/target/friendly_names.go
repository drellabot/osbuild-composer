package target

var targetFriendlyNames = map[TargetName]string{
	TargetNameAWS:              "AWS EC2",
	TargetNameAWSS3:            "AWS S3",
	TargetNameAzure:            "Azure Blob Storage",
	TargetNameAzureImage:       "Azure Image",
	TargetNameGCP:              "Google Cloud",
	TargetNameVMWare:           "VMware vSphere",
	TargetNameContainer:        "Container Registry",
	TargetNameKoji:             "Koji",
	TargetNameOCI:              "OCI",
	TargetNameOCIObjectStorage: "OCI Object Storage",
	TargetNameWorkerServer:     "Worker Server",
}

func FriendlyName(name TargetName) string {
	if friendly, ok := targetFriendlyNames[name]; ok {
		return friendly
	}
	return "target"
}
