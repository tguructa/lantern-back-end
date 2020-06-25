package validation

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/onc-healthit/lantern-back-end/endpointmanager/pkg/capabilityparser"
	"github.com/onc-healthit/lantern-back-end/endpointmanager/pkg/endpointmanager"
)

var tls12 = "TLS 1.2"
var tls13 = "TLS 1.3"

var usCoreProfiles = []string{"AllergyIntolerance", "CarePlan", "CareTeam",
	"Condition", "DiagnosticReport", "DocumentReference", "Encounter", "Goal",
	"Immunization", "Device", "Observation", "Location", "Medication",
	"MedicationRequest", "Organization", "Practitioner", "PractitionerRole",
	"Procedure", "Provenance"}

type r4Validation struct {
	baseVal
}

func newR4Val() *r4Validation {
	return &r4Validation{
		baseVal: baseVal{},
	}
}

func (v *r4Validation) RunValidation(capStat capabilityparser.CapabilityStatement,
	httpResponse int,
	mimeTypes []string,
	fhirVersion string,
	tlsVersion string,
	smartHTTPRsp int) endpointmanager.Validation {
	var validationResults []endpointmanager.Rule
	validationWarnings := make([]endpointmanager.Rule, 0)

	returnedRule := v.CapStatExists(capStat)
	validationResults = append(validationResults, returnedRule)

	returnedRule = v.MimeTypeValid(mimeTypes, fhirVersion)
	validationResults = append(validationResults, returnedRule)

	returnedRule = v.HTTPResponseValid(httpResponse)
	validationResults = append(validationResults, returnedRule)

	returnedRule = v.baseVal.FhirVersion(fhirVersion)
	validationResults = append(validationResults, returnedRule)

	returnedRule = v.TLSVersion(tlsVersion)
	validationResults = append(validationResults, returnedRule)

	returnedRule = v.PatientResourceExists(capStat)
	validationResults = append(validationResults, returnedRule)

	returnedRule = v.OtherResourceExists(capStat)
	validationResults = append(validationResults, returnedRule)

	returnedRule = v.SmartHTTPResponseValid(smartHTTPRsp)
	validationResults = append(validationResults, returnedRule)

	returnedRules := v.KindValid(capStat)
	validationResults = append(validationResults, returnedRules[0], returnedRules[1])

	returnedRule = v.MessagingEndpointValid(capStat)
	validationResults = append(validationResults, returnedRule)

	returnedRule = v.EndpointFunctionValid(capStat)
	validationResults = append(validationResults, returnedRule)

	returnedRule = v.DescribeEndpointValid(capStat)
	validationResults = append(validationResults, returnedRule)

	returnedRule = v.DocumentSetValid(capStat)
	validationResults = append(validationResults, returnedRule)

	returnedRule = v.UniqueResources(capStat)
	validationResults = append(validationResults, returnedRule)

	returnedRule = v.SearchParamsUnique(capStat)
	validationResults = append(validationResults, returnedRule)

	validations := endpointmanager.Validation{
		Results:  validationResults,
		Warnings: validationWarnings,
	}

	return validations
}

func (v *r4Validation) CapStatExists(capStat capabilityparser.CapabilityStatement) endpointmanager.Rule {
	baseRule := v.baseVal.CapStatExists(capStat)
	baseRule.Comment = "Servers SHALL provide a Capability Statement that specifies which interactions and resources are supported."
	baseRule.Reference = "http://hl7.org/fhir/http.html"
	baseRule.ImplGuide = "USCore 3.1"
	return baseRule
}

func (v *r4Validation) MimeTypeValid(mimeTypes []string, fhirVersion string) endpointmanager.Rule {
	baseRule := v.baseVal.MimeTypeValid(mimeTypes, fhirVersion)
	baseRule.Reference = "http://hl7.org/fhir/http.html"
	baseRule.ImplGuide = "USCore 3.1"
	return baseRule
}

func (v *r4Validation) HTTPResponseValid(httpResponse int) endpointmanager.Rule {
	baseRule := v.baseVal.HTTPResponseValid(httpResponse)
	baseRule.Reference = "http://hl7.org/fhir/http.html"
	baseRule.ImplGuide = "USCore 3.1"
	if (httpResponse != 0) && (httpResponse != 200) {
		strResp := strconv.Itoa(httpResponse)
		baseRule.Comment = "The HTTP response code was " + strResp + " instead of 200. Applications SHALL return a resource that describes the functionality of the server end-point."
	}
	return baseRule
}

func (v *r4Validation) TLSVersion(tlsVersion string) endpointmanager.Rule {
	ruleError := endpointmanager.Rule{
		RuleName:  endpointmanager.TLSVersion,
		Valid:     true,
		Expected:  "TLS 1.2, TLS 1.3",
		Actual:    tlsVersion,
		Comment:   "Systems SHALL use TLS version 1.2 or higher for all transmissions not taking place over a secure network connection.",
		Reference: "https://www.hl7.org/fhir/us/core/security.html",
		ImplGuide: "USCore 3.1",
	}

	if (tlsVersion != tls12) && (tlsVersion != tls13) {
		ruleError.Valid = false
	}

	return ruleError
}

func (v *r4Validation) PatientResourceExists(capStat capabilityparser.CapabilityStatement) endpointmanager.Rule {
	baseComment := "The US Core Server SHALL support the US Core Patient resource profile."
	returnVal := checkResourceList(capStat, endpointmanager.PatResourceExists)
	returnVal.Comment = returnVal.Comment + baseComment
	returnVal.Reference = "https://www.hl7.org/fhir/us/core/CapabilityStatement-us-core-server.html"
	return returnVal
}

func (v *r4Validation) OtherResourceExists(capStat capabilityparser.CapabilityStatement) endpointmanager.Rule {
	baseComment := "The US Core Server SHALL support at least one additional resource profile (besides Patient) from the list of US Core Profiles. "
	returnVal := checkResourceList(capStat, endpointmanager.OtherResourceExists)
	returnVal.Comment = returnVal.Comment + baseComment
	returnVal.Reference = "https://www.hl7.org/fhir/us/core/CapabilityStatement-us-core-server.html"
	return returnVal
}

func checkResourceList(capStat capabilityparser.CapabilityStatement, rule endpointmanager.RuleOption) endpointmanager.Rule {
	ruleError := endpointmanager.Rule{
		RuleName:  rule,
		Valid:     true,
		Expected:  "true",
		ImplGuide: "USCore 3.1",
	}

	if capStat == nil {
		ruleError.Valid = false
		ruleError.Actual = "false"
		ruleError.Comment = "The Capability Statement does not exist; cannot check resource profiles. "
		return ruleError
	}

	rest, err := capStat.GetRest()
	if err != nil {
		ruleError.Valid = false
		ruleError.Actual = "false"
		ruleError.Comment = "Rest field does not exist. "
		return ruleError
	}

	var uniqueRecs []string
	for _, restElem := range rest {
		resourceList, err := capStat.GetResourceList(restElem)
		if err != nil || len(resourceList) == 0 {
			ruleError.Valid = false
			ruleError.Actual = "false"
			ruleError.Comment = "The Resource Profiles do not exist. "
			return ruleError
		}
		for _, resource := range resourceList {
			typeVal := resource["type"]
			if typeVal == nil {
				ruleError.Valid = false
				ruleError.Actual = "false"
				ruleError.Comment = "The Resource Profiles are not properly formatted. "
				return ruleError
			}
			typeStr, ok := typeVal.(string)
			if !ok {
				ruleError.Valid = false
				ruleError.Actual = "false"
				ruleError.Comment = "The Resource Profiles are not properly formatted. "
				return ruleError
			}
			if rule == endpointmanager.OtherResourceExists {
				if stringInList(typeStr, usCoreProfiles) {
					return ruleError
				}
			} else if rule == endpointmanager.PatResourceExists {
				if typeStr == "Patient" {
					return ruleError
				}
			} else if rule == endpointmanager.UniqueResourcesRule {
				if stringInList(typeStr, uniqueRecs) {
					ruleError.Valid = false
					ruleError.Actual = "false"
					ruleError.Comment = fmt.Sprintf("The resource type %s is not unique. ", typeStr)
					return ruleError
				}
				uniqueRecs = append(uniqueRecs, typeStr)
			} else if rule == endpointmanager.SearchParamsRule {
				check, err := areSearchParamsValid(resource)
				if err != nil {
					ruleError.Valid = false
					ruleError.Actual = "false"
					ruleError.Comment = ruleError.Comment + fmt.Sprintf("The resource type %s is not formatted properly. ", typeStr)
				} else {
					if !check {
						ruleError.Valid = false
						ruleError.Actual = "false"
						ruleError.Comment = ruleError.Comment + fmt.Sprintf("The resource type %s does not have unique searchParams. ", typeStr)
					}
				}
			}
		}
	}

	if rule == endpointmanager.UniqueResourcesRule || rule == endpointmanager.SearchParamsRule {
		return ruleError
	}
	ruleError.Valid = false
	ruleError.Actual = "false"
	return ruleError
}

func (v *r4Validation) SmartHTTPResponseValid(smartHTTPRsp int) endpointmanager.Rule {
	baseComment := "FHIR endpoints requiring authorization SHALL serve a JSON document at the location formed by appending /.well-known/smart-configuration to their base URL."
	baseRule := v.baseVal.HTTPResponseValid(smartHTTPRsp)
	baseRule.RuleName = endpointmanager.SmartHTTPRespRule
	baseRule.Comment = baseComment
	baseRule.Reference = "http://www.hl7.org/fhir/smart-app-launch/conformance/index.html"
	baseRule.ImplGuide = "USCore 3.1"
	if (smartHTTPRsp != 0) && (smartHTTPRsp != 200) {
		strResp := strconv.Itoa(smartHTTPRsp)
		baseRule.Comment = "The HTTP response code was " + strResp + " instead of 200. Applications SHALL return a resource that describes the functionality of the server end-point. " + baseComment
	}
	return baseRule
}

func (v *r4Validation) KindValid(capStat capabilityparser.CapabilityStatement) []endpointmanager.Rule {
	var rules []endpointmanager.Rule
	baseRule := v.baseVal.KindValid(capStat)
	baseRule[0].Reference = "http://hl7.org/fhir/capabilitystatement.html"
	baseRule[0].ImplGuide = "USCore 3.1"
	rules = append(rules, baseRule[0])

	instanceRule := endpointmanager.Rule{
		RuleName:  endpointmanager.InstanceRule,
		Valid:     true,
		Expected:  "true",
		Actual:    "true",
		Comment:   "If kind = instance, implementation must be present. This endpoint must be an instance.",
		Reference: "http://hl7.org/fhir/capabilitystatement.html",
		ImplGuide: "USCore 3.1",
	}
	impl, err := capStat.GetImplementation()
	if err != nil || len(impl) == 0 {
		instanceRule.Valid = false
		instanceRule.Actual = "false"
	}
	rules = append(rules, instanceRule)
	return rules
}

// MessagingEndpointValid checks the requirement "Messaging endpoint is required (and is only permitted) when a statement is for an implementation."
// Every endpoint we are testing should be an implementation, which means the endpoint field should be there.
func (v *r4Validation) MessagingEndpointValid(capStat capabilityparser.CapabilityStatement) endpointmanager.Rule {
	baseComment := "Messaging end-point is required (and is only permitted) when a statement is for an implementation. This endpoint must be an implementation."
	ruleError := endpointmanager.Rule{
		RuleName:  endpointmanager.MessagingEndptRule,
		Valid:     true,
		Expected:  "true",
		Actual:    "true",
		Comment:   baseComment,
		Reference: "http://hl7.org/fhir/capabilitystatement.html",
		ImplGuide: "USCore 3.1",
	}

	kindRule := v.baseVal.KindValid(capStat)
	if !kindRule[0].Valid {
		ruleError.Valid = false
		ruleError.Actual = "false"
		ruleError.Comment = kindRule[0].Comment + " " + baseComment
		return ruleError
	}
	messaging, err := capStat.GetMessaging()
	if err != nil {
		ruleError.Valid = false
		ruleError.Actual = "false"
		ruleError.Comment = "Messaging does not exist. " + baseComment
		return ruleError
	}
	for _, message := range messaging {
		endpoints, err := capStat.GetMessagingEndpoint(message)
		if err != nil || len(endpoints) == 0 {
			ruleError.Valid = false
			ruleError.Actual = "false"
			ruleError.Comment = "Endpoint field in Messaging does not exist. " + baseComment
			return ruleError
		}
	}

	return ruleError
}

// EndpointFunctionValid checks the requirement "A Capability Statement SHALL have at least one of REST,
// messaging or document element."
func (v *r4Validation) EndpointFunctionValid(capStat capabilityparser.CapabilityStatement) endpointmanager.Rule {
	var actualVal []string
	baseComment := "A Capability Statement SHALL have at least one of REST, messaging or document element."
	ruleError := endpointmanager.Rule{
		RuleName:  endpointmanager.EndptFunctionRule,
		Valid:     true,
		Expected:  "rest OR messaging OR document",
		Comment:   baseComment,
		Reference: "http://hl7.org/fhir/capabilitystatement.html",
		ImplGuide: "USCore 3.1",
	}
	// If rest is not nil, add to actual list
	rest, err := capStat.GetRest()
	if err == nil && len(rest) > 0 {
		actualVal = append(actualVal, "rest")
	}
	// If messaging is not nil, add to actual list
	messaging, err := capStat.GetMessaging()
	if err == nil && len(messaging) > 0 {
		actualVal = append(actualVal, "messaging")
	}
	// if document is not nil, add to actual list
	document, err := capStat.GetDocument()
	if err == nil && len(document) > 0 {
		actualVal = append(actualVal, "document")
	}
	// If none of the above exist, the capability statement is not valid
	if len(actualVal) == 0 {
		ruleError.Actual = ""
		ruleError.Valid = false
		return ruleError
	}
	ruleError.Actual = strings.Join(actualVal, ",")
	return ruleError
}

// DescribeEndpointValid checks the requirement: "A Capability Statement SHALL have at least one of description,
// software, or implementation element."
func (v *r4Validation) DescribeEndpointValid(capStat capabilityparser.CapabilityStatement) endpointmanager.Rule {
	var actualVal []string
	baseComment := "A Capability Statement SHALL have at least one of description, software, or implementation element."
	ruleError := endpointmanager.Rule{
		RuleName:  endpointmanager.DescribeEndptRule,
		Valid:     true,
		Expected:  "description OR software OR implementation",
		Comment:   baseComment,
		Reference: "http://hl7.org/fhir/capabilitystatement.html",
		ImplGuide: "USCore 3.1",
	}
	// If description is not an empty string, add to actual list
	description, err := capStat.GetDescription()
	if err == nil && len(description) > 0 {
		actualVal = append(actualVal, "description")
	}
	// If software is not nil, add to actual list
	software, err := capStat.GetSoftware()
	if err == nil && len(software) > 0 {
		actualVal = append(actualVal, "software")
	}
	// if implementation is not nil, add to actual list
	implementation, err := capStat.GetImplementation()
	if err == nil && len(implementation) > 0 {
		actualVal = append(actualVal, "implementation")
	}
	// If none of the above exist, the capability statement is not valid
	if len(actualVal) == 0 {
		ruleError.Actual = ""
		ruleError.Valid = false
		return ruleError
	}
	ruleError.Actual = strings.Join(actualVal, ",")
	return ruleError
}

// DocumentSetValid checks the requirement: "The set of documents must be unique by the combination of profile and mode."
func (v *r4Validation) DocumentSetValid(capStat capabilityparser.CapabilityStatement) endpointmanager.Rule {
	baseComment := "The set of documents must be unique by the combination of profile and mode."
	ruleError := endpointmanager.Rule{
		RuleName:  endpointmanager.DocumentValidRule,
		Valid:     true,
		Expected:  "true",
		Actual:    "true",
		Comment:   baseComment,
		Reference: "http://hl7.org/fhir/capabilitystatement.html",
		ImplGuide: "USCore 3.1",
	}
	document, err := capStat.GetDocument()
	if err != nil {
		ruleError.Valid = false
		ruleError.Actual = "false"
		ruleError.Comment = "Document field is not formatted correctly. Cannot check if the set of documents are unique. " + baseComment
		return ruleError
	}
	if err == nil && len(document) == 0 {
		ruleError.Comment = "Document field does not exist."
		return ruleError
	}
	var uniqueIDs []string
	invalid := false
	for _, doc := range document {
		mode := doc["mode"]
		if mode == nil {
			invalid = true
			break
		}
		modeStr, ok := mode.(string)
		if !ok {
			invalid = true
			break
		}
		profile := doc["profile"]
		if profile == nil {
			invalid = true
			break
		}
		profileStr, ok := profile.(string)
		if !ok {
			invalid = true
			break
		}
		id := profileStr + "." + modeStr
		if stringInList(id, uniqueIDs) {
			ruleError.Valid = false
			ruleError.Actual = "false"
			ruleError.Comment = "The set of documents are not unique. " + baseComment
			return ruleError
		}
		uniqueIDs = append(uniqueIDs, id)
	}
	if invalid {
		ruleError.Valid = false
		ruleError.Actual = "false"
		ruleError.Comment = "Document field is not formatted correctly. Cannot check if the set of documents are unique. " + baseComment
		return ruleError
	}
	return ruleError
}

// UniqueResources checks the requirement: "A given resource can only be described once per RESTful mode."
func (v *r4Validation) UniqueResources(capStat capabilityparser.CapabilityStatement) endpointmanager.Rule {
	baseComment := "A given resource can only be described once per RESTful mode."
	returnVal := checkResourceList(capStat, endpointmanager.UniqueResourcesRule)
	returnVal.Comment = returnVal.Comment + baseComment
	returnVal.Reference = "http://hl7.org/fhir/capabilitystatement.html"
	return returnVal
}

// SearchParamsUnique checks the requirement: "Search parameter names must be unique in the context of a resource."
func (v *r4Validation) SearchParamsUnique(capStat capabilityparser.CapabilityStatement) endpointmanager.Rule {
	baseComment := "Search parameter names must be unique in the context of a resource."
	returnVal := checkResourceList(capStat, endpointmanager.SearchParamsRule)
	returnVal.Comment = returnVal.Comment + baseComment
	returnVal.Reference = "http://hl7.org/fhir/capabilitystatement.html"
	return returnVal
}

func areSearchParamsValid(resource map[string]interface{}) (bool, error) {
	var searchParams []string
	search := resource["searchParam"]
	if search == nil {
		return true, nil
	}
	searchList, ok := search.([]interface{})
	if !ok {
		return false, fmt.Errorf("Unable to cast searchParam value in a resource to a list")
	}
	for _, elem := range searchList {
		obj, ok := elem.(map[string]interface{})
		if !ok {
			return false, fmt.Errorf("Unable to cast element of searchParam list to a map[string]interface{}")
		}
		name := obj["name"]
		if name == nil {
			return false, fmt.Errorf("Name does not exist but is required in searchParam values")
		}
		nameStr, ok := obj["name"].(string)
		if !ok {
			return false, fmt.Errorf("Unable to cast the name of a searchParam to a string")
		}
		if stringInList(nameStr, searchParams) {
			return false, nil
		}
		searchParams = append(searchParams, nameStr)
	}
	return true, nil
}

func stringInList(str string, list []string) bool {
	for _, b := range list {
		if b == str {
			return true
		}
	}
	return false
}
