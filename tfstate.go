package stefunny

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sort"
	"strings"

	"github.com/fujiwara/tfstate-lookup/tfstate"
)

// ListResourcesFromTFState returns resource arn, security group id, vpc id, subnet id, and caller account_id
func ListResourcesFromTFState(ctx context.Context, loc string) (*OrderdMap[string, string], error) {
	s, err := tfstate.ReadURL(ctx, loc)
	if err != nil {
		return nil, fmt.Errorf("failed to read tfstate: %w", err)
	}
	list, err := s.List()
	if err != nil {
		return nil, fmt.Errorf("failed to list resources: %w", err)
	}
	resources, resourceValues := newResourcesReverseMapFromTFState(s, list)
	orderd := NewOrderdMap[string, string]()
	sort.Slice(resourceValues, func(i, j int) bool {
		return len(resourceValues[i]) < len(resourceValues[j])
	})
	for _, v := range resourceValues {
		orderd.Set(resources[v], v)
	}
	return orderd, nil
}

func newResourcesReverseMapFromTFState(s *tfstate.TFState, list []string) (map[string]string, []string) {
	log.Println("[debug] start list resources from tfstate")
	resources := make(map[string]string) // resource value => lookup key
	resourceValues := make([]string, 0)
	for _, r := range list {
		if !strings.HasPrefix(r, "aws_") && strings.HasPrefix(r, "data.aws_") {
			log.Printf("[debug] skip `%s`, this is not aws resource", r)
			continue
		}
		obj, err := s.Lookup(r)
		if err != nil {
			log.Printf("[warn] skip `%s`, cannot lookup: %v", r, err)
			continue
		}
		bs, err := json.Marshal(obj.Value)
		if err != nil {
			log.Printf("[warn] skip `%s`, cannot marshal object as json: %v", r, err)
			continue
		}
		var data map[string]interface{}
		if err := json.Unmarshal(bs, &data); err != nil {
			log.Printf("[debug] skip `%s`, cannot unmarshal json: %v", r, err)
			continue
		}
		reverseMap, ok := lookupResourceKey(r, data)
		if !ok {
			log.Printf("[debug] skip `%s`, cannot lookup resource key", r)
			continue
		}
		for value, key := range reverseMap {
			if duplicated, ok := resources[value]; ok {
				log.Printf("[warn] `%s` is duplicated (`%s` and `%s`), skip after key", value, duplicated, key)
				continue
			}
			resources[value] = key
			resourceValues = append(resourceValues, value)
		}
	}
	return resources, resourceValues
}

var (
	specifiedResources = map[string][]string{
		"aws_security_group": {"id"},
		"aws_vpc":            {"id"},
		"aws_subnet":         {"id"},
		"aws_s3_bucket":      {"bucket"},
	}
	ignoreResourceSuffix = []string{
		"_policy",
	}
)

func lookupResourceKey(resourcePrefix string, data map[string]any) (map[string]string, bool) {
	parts := strings.Split(resourcePrefix, ".")
	for _, suffix := range ignoreResourceSuffix {
		if strings.HasSuffix(parts[0], suffix) {
			log.Printf("[debug] ignore `%s` reason is has suffix `%s`", resourcePrefix, suffix)
			return nil, false
		}
	}
	if strings.HasPrefix("data.aws_caller_identity.", resourcePrefix) {
		accountID, ok := data["account_id"].(string)
		if !ok {
			log.Printf("[debug] `%s.account_id` is not found or not string: acutal `%T`", resourcePrefix, data["account_id"])
			return nil, false
		}
		return map[string]string{
			accountID: fmt.Sprintf("%s.account_id", resourcePrefix),
		}, true
	}
	reverceMap := make(map[string]string)
	for prefix, attrs := range specifiedResources {
		if !strings.HasPrefix(resourcePrefix, prefix+".") {
			continue
		}
		for _, attr := range attrs {
			key := fmt.Sprintf("%s.%s", resourcePrefix, attr)
			v, ok := data[attr]
			if !ok {
				log.Printf("[warn] `%s` detect but not found `%s`", resourcePrefix, attr)
				continue
			}
			attrStr, ok := v.(string)
			if !ok {
				log.Printf("[warn] `%s.%s` is not string: acutal `%T`", resourcePrefix, attr, v)
				continue
			}
			reverceMap[attrStr] = key
			log.Printf("[debug] `%s.%s` is `%s`", resourcePrefix, attr, attrStr)
		}
		break
	}
	if arn, ok := data["arn"].(string); ok {
		reverceMap[arn] = fmt.Sprintf("%s.arn", resourcePrefix)
		log.Printf("[debug] `%s.arn` is `%s`", resourcePrefix, arn)
	}
	if quarifiedArn, ok := data["qualified_arn"].(string); ok {
		reverceMap[quarifiedArn] = fmt.Sprintf("%s.qualified_arn", resourcePrefix)
		log.Printf("[debug] `%s.qualified_arn` is `%s`", resourcePrefix, quarifiedArn)
	}
	if uri, ok := data["uri"].(string); ok {
		reverceMap[uri] = fmt.Sprintf("%s.uri", resourcePrefix)
		log.Printf("[debug] `%s.uri` is `%s`", resourcePrefix, uri)
	}
	return reverceMap, len(reverceMap) > 0
}
