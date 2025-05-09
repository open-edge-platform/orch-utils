// SPDX-FileCopyrightText: 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0
// Code generated by github.com/vmware-tanzu/graph-framework-for-microservices/gqlgen, DO NOT EDIT.

package model

type NexusGraphqlResponse struct {
	Code         *int    `json:"Code"`
	Message      *string `json:"Message"`
	Data         *string `json:"Data"`
	Last         *string `json:"Last"`
	TotalRecords *int    `json:"TotalRecords"`
}

type TimeSeriesData struct {
	Code         *int    `json:"Code"`
	Message      *string `json:"Message"`
	Data         *string `json:"Data"`
	Last         *string `json:"Last"`
	TotalRecords *int    `json:"TotalRecords"`
}

type ConfigConfig struct {
	Id                *string                         `json:"Id"`
	ParentLabels      map[string]interface{}          `json:"ParentLabels"`
	QueryExample      *NexusGraphqlResponse           `json:"QueryExample"`
	ACPPolicies       []*PolicypkgAccessControlPolicy `json:"ACPPolicies"`
	FooExample        []*ConfigFooTypeABC             `json:"FooExample"`
	MyStr0            *string                         `json:"MyStr0"`
	MyStr1            *string                         `json:"MyStr1"`
	MyStr2            *string                         `json:"MyStr2"`
	XYZPort           *string                         `json:"XYZPort"`
	ABCHost           *string                         `json:"ABCHost"`
	ClusterNamespaces *string                         `json:"ClusterNamespaces"`
	TestValMarkers    *string                         `json:"TestValMarkers"`
	Instance          *float64                        `json:"Instance"`
	CuOption          *string                         `json:"CuOption"`
	GNS               *GnsGns                         `json:"GNS"`
	DNS               *GnsDns                         `json:"DNS"`
	VMPPolicies       *PolicypkgVMpolicy              `json:"VMPPolicies"`
	Domain            *ConfigDomain                   `json:"Domain"`
	SvcGrpInfo        *ServicegroupSvcGroupLinkInfo   `json:"SvcGrpInfo"`
}

type ConfigDomain struct {
	Id               *string                `json:"Id"`
	ParentLabels     map[string]interface{} `json:"ParentLabels"`
	PointPort        *string                `json:"PointPort"`
	PointString      *string                `json:"PointString"`
	PointInt         *int                   `json:"PointInt"`
	PointMap         *string                `json:"PointMap"`
	PointSlice       *string                `json:"PointSlice"`
	SliceOfPoints    *string                `json:"SliceOfPoints"`
	SliceOfArrPoints *string                `json:"SliceOfArrPoints"`
	MapOfArrsPoints  *string                `json:"MapOfArrsPoints"`
	PointStruct      *string                `json:"PointStruct"`
}

type ConfigFooTypeABC struct {
	Id           *string                `json:"Id"`
	ParentLabels map[string]interface{} `json:"ParentLabels"`
	FooA         *string                `json:"FooA"`
	FooB         *string                `json:"FooB"`
	FooD         *string                `json:"FooD"`
	FooF         *string                `json:"FooF"`
}

type GnsBarChild struct {
	Id           *string                `json:"Id"`
	ParentLabels map[string]interface{} `json:"ParentLabels"`
	Name         *string                `json:"Name"`
}

type GnsDns struct {
	Id           *string                `json:"Id"`
	ParentLabels map[string]interface{} `json:"ParentLabels"`
}

type GnsGns struct {
	Id                               *string                                           `json:"Id"`
	ParentLabels                     map[string]interface{}                            `json:"ParentLabels"`
	queryGns1                        *NexusGraphqlResponse                             `json:"queryGns1"`
	queryGnsQM1                      *TimeSeriesData                                   `json:"queryGnsQM1"`
	queryGnsQM                       *TimeSeriesData                                   `json:"queryGnsQM"`
	TestOptionalParentPathParamChild []*OptionalparentpathparamOptionalParentPathParam `json:"TestOptionalParentPathParamChild"`
	Domain                           *string                                           `json:"Domain"`
	UseSharedGateway                 *bool                                             `json:"UseSharedGateway"`
	Annotations                      *string                                           `json:"Annotations"`
	TargetPort                       *string                                           `json:"TargetPort"`
	Description                      *string                                           `json:"Description"`
	Meta                             *string                                           `json:"Meta"`
	IntOrString                      *string                                           `json:"IntOrString"`
	Port                             *int                                              `json:"Port"`
	OtherDescription                 *string                                           `json:"OtherDescription"`
	MapPointer                       *string                                           `json:"MapPointer"`
	SlicePointer                     *string                                           `json:"SlicePointer"`
	WorkloadSpec                     *string                                           `json:"WorkloadSpec"`
	DifferentSpec                    *string                                           `json:"DifferentSpec"`
	ServiceSegmentRef                *string                                           `json:"ServiceSegmentRef"`
	ServiceSegmentRefPointer         *string                                           `json:"ServiceSegmentRefPointer"`
	ServiceSegmentRefs               *string                                           `json:"ServiceSegmentRefs"`
	ServiceSegmentRefMap             *string                                           `json:"ServiceSegmentRefMap"`
	GnsAccessControlPolicy           *PolicypkgAccessControlPolicy                     `json:"GnsAccessControlPolicy"`
	FooChild                         *GnsBarChild                                      `json:"FooChild"`
}

type GnsIgnoreChild struct {
	Id           *string                `json:"Id"`
	ParentLabels map[string]interface{} `json:"ParentLabels"`
	Name         *string                `json:"Name"`
}

type OptionalparentpathparamOptionalParentPathParam struct {
	Id           *string                `json:"Id"`
	ParentLabels map[string]interface{} `json:"ParentLabels"`
}

type PolicypkgACPConfig struct {
	Id           *string                `json:"Id"`
	ParentLabels map[string]interface{} `json:"ParentLabels"`
	DisplayName  *string                `json:"DisplayName"`
	Gns          *string                `json:"Gns"`
	Description  *string                `json:"Description"`
	Tags         *string                `json:"Tags"`
	ProjectId    *string                `json:"ProjectId"`
	Conditions   *string                `json:"Conditions"`
}

type PolicypkgAccessControlPolicy struct {
	Id            *string                `json:"Id"`
	ParentLabels  map[string]interface{} `json:"ParentLabels"`
	PolicyConfigs []*PolicypkgACPConfig  `json:"PolicyConfigs"`
}

type PolicypkgVMpolicy struct {
	Id           *string                `json:"Id"`
	ParentLabels map[string]interface{} `json:"ParentLabels"`
	queryGns1    *NexusGraphqlResponse  `json:"queryGns1"`
	queryGnsQM1  *TimeSeriesData        `json:"queryGnsQM1"`
	queryGnsQM   *TimeSeriesData        `json:"queryGnsQM"`
}

type RootRoot struct {
	Id           *string                `json:"Id"`
	ParentLabels map[string]interface{} `json:"ParentLabels"`
	Config       *ConfigConfig          `json:"Config"`
}

type ServicegroupSvcGroupLinkInfo struct {
	Id           *string                `json:"Id"`
	ParentLabels map[string]interface{} `json:"ParentLabels"`
	ClusterName  *string                `json:"ClusterName"`
	DomainName   *string                `json:"DomainName"`
	ServiceName  *string                `json:"ServiceName"`
	ServiceType  *string                `json:"ServiceType"`
}
