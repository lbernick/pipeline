// Code generated by smithy-go-codegen DO NOT EDIT.

package types

type EncryptionType string

// Enum values for EncryptionType
const (
	EncryptionTypeAes256 EncryptionType = "AES256"
	EncryptionTypeKms    EncryptionType = "KMS"
)

// Values returns all known values for EncryptionType. Note that this can be
// expanded in the future, and so it is only as up to date as the client. The
// ordering of this slice is not guaranteed to be stable across updates.
func (EncryptionType) Values() []EncryptionType {
	return []EncryptionType{
		"AES256",
		"KMS",
	}
}

type FindingSeverity string

// Enum values for FindingSeverity
const (
	FindingSeverityInformational FindingSeverity = "INFORMATIONAL"
	FindingSeverityLow           FindingSeverity = "LOW"
	FindingSeverityMedium        FindingSeverity = "MEDIUM"
	FindingSeverityHigh          FindingSeverity = "HIGH"
	FindingSeverityCritical      FindingSeverity = "CRITICAL"
	FindingSeverityUndefined     FindingSeverity = "UNDEFINED"
)

// Values returns all known values for FindingSeverity. Note that this can be
// expanded in the future, and so it is only as up to date as the client. The
// ordering of this slice is not guaranteed to be stable across updates.
func (FindingSeverity) Values() []FindingSeverity {
	return []FindingSeverity{
		"INFORMATIONAL",
		"LOW",
		"MEDIUM",
		"HIGH",
		"CRITICAL",
		"UNDEFINED",
	}
}

type ImageActionType string

// Enum values for ImageActionType
const (
	ImageActionTypeExpire ImageActionType = "EXPIRE"
)

// Values returns all known values for ImageActionType. Note that this can be
// expanded in the future, and so it is only as up to date as the client. The
// ordering of this slice is not guaranteed to be stable across updates.
func (ImageActionType) Values() []ImageActionType {
	return []ImageActionType{
		"EXPIRE",
	}
}

type ImageFailureCode string

// Enum values for ImageFailureCode
const (
	ImageFailureCodeInvalidImageDigest            ImageFailureCode = "InvalidImageDigest"
	ImageFailureCodeInvalidImageTag               ImageFailureCode = "InvalidImageTag"
	ImageFailureCodeImageTagDoesNotMatchDigest    ImageFailureCode = "ImageTagDoesNotMatchDigest"
	ImageFailureCodeImageNotFound                 ImageFailureCode = "ImageNotFound"
	ImageFailureCodeMissingDigestAndTag           ImageFailureCode = "MissingDigestAndTag"
	ImageFailureCodeImageReferencedByManifestList ImageFailureCode = "ImageReferencedByManifestList"
	ImageFailureCodeKmsError                      ImageFailureCode = "KmsError"
)

// Values returns all known values for ImageFailureCode. Note that this can be
// expanded in the future, and so it is only as up to date as the client. The
// ordering of this slice is not guaranteed to be stable across updates.
func (ImageFailureCode) Values() []ImageFailureCode {
	return []ImageFailureCode{
		"InvalidImageDigest",
		"InvalidImageTag",
		"ImageTagDoesNotMatchDigest",
		"ImageNotFound",
		"MissingDigestAndTag",
		"ImageReferencedByManifestList",
		"KmsError",
	}
}

type ImageTagMutability string

// Enum values for ImageTagMutability
const (
	ImageTagMutabilityMutable   ImageTagMutability = "MUTABLE"
	ImageTagMutabilityImmutable ImageTagMutability = "IMMUTABLE"
)

// Values returns all known values for ImageTagMutability. Note that this can be
// expanded in the future, and so it is only as up to date as the client. The
// ordering of this slice is not guaranteed to be stable across updates.
func (ImageTagMutability) Values() []ImageTagMutability {
	return []ImageTagMutability{
		"MUTABLE",
		"IMMUTABLE",
	}
}

type LayerAvailability string

// Enum values for LayerAvailability
const (
	LayerAvailabilityAvailable   LayerAvailability = "AVAILABLE"
	LayerAvailabilityUnavailable LayerAvailability = "UNAVAILABLE"
)

// Values returns all known values for LayerAvailability. Note that this can be
// expanded in the future, and so it is only as up to date as the client. The
// ordering of this slice is not guaranteed to be stable across updates.
func (LayerAvailability) Values() []LayerAvailability {
	return []LayerAvailability{
		"AVAILABLE",
		"UNAVAILABLE",
	}
}

type LayerFailureCode string

// Enum values for LayerFailureCode
const (
	LayerFailureCodeInvalidLayerDigest LayerFailureCode = "InvalidLayerDigest"
	LayerFailureCodeMissingLayerDigest LayerFailureCode = "MissingLayerDigest"
)

// Values returns all known values for LayerFailureCode. Note that this can be
// expanded in the future, and so it is only as up to date as the client. The
// ordering of this slice is not guaranteed to be stable across updates.
func (LayerFailureCode) Values() []LayerFailureCode {
	return []LayerFailureCode{
		"InvalidLayerDigest",
		"MissingLayerDigest",
	}
}

type LifecyclePolicyPreviewStatus string

// Enum values for LifecyclePolicyPreviewStatus
const (
	LifecyclePolicyPreviewStatusInProgress LifecyclePolicyPreviewStatus = "IN_PROGRESS"
	LifecyclePolicyPreviewStatusComplete   LifecyclePolicyPreviewStatus = "COMPLETE"
	LifecyclePolicyPreviewStatusExpired    LifecyclePolicyPreviewStatus = "EXPIRED"
	LifecyclePolicyPreviewStatusFailed     LifecyclePolicyPreviewStatus = "FAILED"
)

// Values returns all known values for LifecyclePolicyPreviewStatus. Note that
// this can be expanded in the future, and so it is only as up to date as the
// client. The ordering of this slice is not guaranteed to be stable across
// updates.
func (LifecyclePolicyPreviewStatus) Values() []LifecyclePolicyPreviewStatus {
	return []LifecyclePolicyPreviewStatus{
		"IN_PROGRESS",
		"COMPLETE",
		"EXPIRED",
		"FAILED",
	}
}

type ReplicationStatus string

// Enum values for ReplicationStatus
const (
	ReplicationStatusInProgress ReplicationStatus = "IN_PROGRESS"
	ReplicationStatusComplete   ReplicationStatus = "COMPLETE"
	ReplicationStatusFailed     ReplicationStatus = "FAILED"
)

// Values returns all known values for ReplicationStatus. Note that this can be
// expanded in the future, and so it is only as up to date as the client. The
// ordering of this slice is not guaranteed to be stable across updates.
func (ReplicationStatus) Values() []ReplicationStatus {
	return []ReplicationStatus{
		"IN_PROGRESS",
		"COMPLETE",
		"FAILED",
	}
}

type RepositoryFilterType string

// Enum values for RepositoryFilterType
const (
	RepositoryFilterTypePrefixMatch RepositoryFilterType = "PREFIX_MATCH"
)

// Values returns all known values for RepositoryFilterType. Note that this can be
// expanded in the future, and so it is only as up to date as the client. The
// ordering of this slice is not guaranteed to be stable across updates.
func (RepositoryFilterType) Values() []RepositoryFilterType {
	return []RepositoryFilterType{
		"PREFIX_MATCH",
	}
}

type ScanFrequency string

// Enum values for ScanFrequency
const (
	ScanFrequencyScanOnPush     ScanFrequency = "SCAN_ON_PUSH"
	ScanFrequencyContinuousScan ScanFrequency = "CONTINUOUS_SCAN"
	ScanFrequencyManual         ScanFrequency = "MANUAL"
)

// Values returns all known values for ScanFrequency. Note that this can be
// expanded in the future, and so it is only as up to date as the client. The
// ordering of this slice is not guaranteed to be stable across updates.
func (ScanFrequency) Values() []ScanFrequency {
	return []ScanFrequency{
		"SCAN_ON_PUSH",
		"CONTINUOUS_SCAN",
		"MANUAL",
	}
}

type ScanningConfigurationFailureCode string

// Enum values for ScanningConfigurationFailureCode
const (
	ScanningConfigurationFailureCodeRepositoryNotFound ScanningConfigurationFailureCode = "REPOSITORY_NOT_FOUND"
)

// Values returns all known values for ScanningConfigurationFailureCode. Note that
// this can be expanded in the future, and so it is only as up to date as the
// client. The ordering of this slice is not guaranteed to be stable across
// updates.
func (ScanningConfigurationFailureCode) Values() []ScanningConfigurationFailureCode {
	return []ScanningConfigurationFailureCode{
		"REPOSITORY_NOT_FOUND",
	}
}

type ScanningRepositoryFilterType string

// Enum values for ScanningRepositoryFilterType
const (
	ScanningRepositoryFilterTypeWildcard ScanningRepositoryFilterType = "WILDCARD"
)

// Values returns all known values for ScanningRepositoryFilterType. Note that
// this can be expanded in the future, and so it is only as up to date as the
// client. The ordering of this slice is not guaranteed to be stable across
// updates.
func (ScanningRepositoryFilterType) Values() []ScanningRepositoryFilterType {
	return []ScanningRepositoryFilterType{
		"WILDCARD",
	}
}

type ScanStatus string

// Enum values for ScanStatus
const (
	ScanStatusInProgress             ScanStatus = "IN_PROGRESS"
	ScanStatusComplete               ScanStatus = "COMPLETE"
	ScanStatusFailed                 ScanStatus = "FAILED"
	ScanStatusUnsupportedImage       ScanStatus = "UNSUPPORTED_IMAGE"
	ScanStatusActive                 ScanStatus = "ACTIVE"
	ScanStatusPending                ScanStatus = "PENDING"
	ScanStatusScanEligibilityExpired ScanStatus = "SCAN_ELIGIBILITY_EXPIRED"
	ScanStatusFindingsUnavailable    ScanStatus = "FINDINGS_UNAVAILABLE"
)

// Values returns all known values for ScanStatus. Note that this can be expanded
// in the future, and so it is only as up to date as the client. The ordering of
// this slice is not guaranteed to be stable across updates.
func (ScanStatus) Values() []ScanStatus {
	return []ScanStatus{
		"IN_PROGRESS",
		"COMPLETE",
		"FAILED",
		"UNSUPPORTED_IMAGE",
		"ACTIVE",
		"PENDING",
		"SCAN_ELIGIBILITY_EXPIRED",
		"FINDINGS_UNAVAILABLE",
	}
}

type ScanType string

// Enum values for ScanType
const (
	ScanTypeBasic    ScanType = "BASIC"
	ScanTypeEnhanced ScanType = "ENHANCED"
)

// Values returns all known values for ScanType. Note that this can be expanded in
// the future, and so it is only as up to date as the client. The ordering of this
// slice is not guaranteed to be stable across updates.
func (ScanType) Values() []ScanType {
	return []ScanType{
		"BASIC",
		"ENHANCED",
	}
}

type TagStatus string

// Enum values for TagStatus
const (
	TagStatusTagged   TagStatus = "TAGGED"
	TagStatusUntagged TagStatus = "UNTAGGED"
	TagStatusAny      TagStatus = "ANY"
)

// Values returns all known values for TagStatus. Note that this can be expanded
// in the future, and so it is only as up to date as the client. The ordering of
// this slice is not guaranteed to be stable across updates.
func (TagStatus) Values() []TagStatus {
	return []TagStatus{
		"TAGGED",
		"UNTAGGED",
		"ANY",
	}
}
