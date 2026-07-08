package storcli

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode"

	"github.com/ps78674/zabbix-raidstat/plugins/internal/functions"
)

// GetControllersIDs - get number of controllers in the system
func GetControllersIDs(execPath string) []string {
	inputData := getJSONCommandOutput(execPath, "show")

	if root, ok := decodeJSON(inputData); ok {
		if data := getControllersIDsFromJSON(root); len(data) > 0 {
			return data
		}
	}

	return getControllersIDsFromText(inputData)
}

// GetLogicalDrivesIDs - get number of logical drives for controller with ID 'controllerID'
func GetLogicalDrivesIDs(execPath string, controllerID string) []string {
	inputData := getJSONCommandOutput(execPath, fmt.Sprintf("/c%s/vall", controllerID), "show")

	if root, ok := decodeJSON(inputData); ok {
		if data := getLogicalDrivesIDsFromJSON(root); len(data) > 0 {
			return data
		}
	}

	return getLogicalDrivesIDsFromText(inputData)
}

// GetPhysicalDrivesIDs - get number of physical drives for controller with ID 'controllerID'
func GetPhysicalDrivesIDs(execPath string, controllerID string) []string {
	inputData := getJSONCommandOutput(execPath, fmt.Sprintf("/c%s/eall/sall", controllerID), "show")

	if root, ok := decodeJSON(inputData); ok {
		if data := getPhysicalDrivesIDsFromJSON(root); len(data) > 0 {
			return data
		}
	}

	return getPhysicalDrivesIDsFromText(inputData)
}

// GetControllerStatus - get controller status
func GetControllerStatus(execPath string, controllerID string, indent int) []byte {
	type ReturnData struct {
		Status        string `json:"status"`
		Model         string `json:"model"`
		BatteryStatus string `json:"batterystatus"`
		CacheStatus   string `json:"cachestatus"`
		Temperature   string `json:"temperature"`
	}

	inputData := getJSONCommandOutput(execPath, fmt.Sprintf("/c%s", controllerID), "show", "all")

	status := ""
	model := ""
	batteryStatus := ""
	cacheStatus := ""
	temperature := ""

	if root, ok := decodeJSON(inputData); ok {
		status = normalizeControllerStatus(firstJSONValue(root,
			"Controller Status",
			"Controller Health",
			"Health",
			"Hlth",
		))
		model = controllerModelFromJSON(root)
		batteryStatus = normalizeBatteryStatus(firstJSONValue(root,
			"BBU Status",
			"Battery Status",
			"Battery State",
			"Battery/Capacitor Status",
			"CacheVault State",
			"CV Status",
			"Energy Pack Status",
		))
		cacheStatus = normalizeCacheStatus(firstJSONValue(root,
			"Cache Status",
			"Any Offline VD Cache Preserved",
			"CacheVault State",
			"CV Status",
			"Energy Pack Status",
		))
		temperature = normalizeTemperature(firstJSONValue(root,
			"ROC temperature(Degree Celsius)",
			"ROC temperature",
			"Controller Temperature",
			"Temperature",
		))
	}

	if model == "" {
		model = textValue(inputData, "Model", "Product Name", "Controller Model")
	}

	if status == "" {
		status = normalizeControllerStatus(textValue(inputData, "Controller Status", "Controller Health"))
	}

	if status == "" {
		status = summarizeControllerStatus(inputData)
	}

	if batteryStatus == "" {
		batteryStatus = normalizeBatteryStatus(textValue(inputData,
			"BBU Status",
			"Battery Status",
			"Battery State",
			"Battery/Capacitor Status",
			"CacheVault State",
			"CV Status",
			"Energy Pack Status",
		))
	}

	if batteryStatus == "" {
		batteryStatus = normalizeBatteryStatus(textValue(inputData, "BBU", "BatteryFRU"))
	}

	if cacheStatus == "" {
		cacheStatus = normalizeCacheStatus(textValue(inputData,
			"Cache Status",
			"Any Offline VD Cache Preserved",
			"CacheVault State",
			"CV Status",
			"Energy Pack Status",
		))
	}

	if cacheStatus == "" && status == "OK" {
		cacheStatus = "OK"
	}

	if temperature == "" {
		temperature = normalizeTemperature(textValue(inputData,
			"ROC temperature(Degree Celsius)",
			"ROC temperature",
			"Controller Temperature",
			"Temperature",
		))
	}

	data := ReturnData{
		Status:        trim(status),
		Model:         trim(model),
		BatteryStatus: trim(batteryStatus),
		CacheStatus:   trim(cacheStatus),
		Temperature:   trim(temperature),
	}

	return append(functions.MarshallJSON(data, indent), "\n"...)
}

// GetLDStatus - get logical drive status
func GetLDStatus(execPath string, controllerID string, deviceID string, indent int) []byte {
	type ReturnData struct {
		Status   string `json:"status"`
		Size     string `json:"size"`
		RaidMode string `json:"raidmode"`
		Name     string `json:"name"`
	}

	inputData := getJSONCommandOutput(execPath, fmt.Sprintf("/c%s/v%s", controllerID, deviceID), "show", "all")

	status := ""
	size := ""
	raidMode := ""
	name := ""

	if root, ok := decodeJSON(inputData); ok {
		if row := findLogicalDriveRow(root, deviceID); len(row) > 0 {
			status = normalizeLogicalDriveStatus(rowValue(row, "State", "Status"))
			size = rowValue(row, "Size")
			raidMode = rowValue(row, "TYPE", "Type", "RAID Type")
			name = rowValue(row, "Name")
		}

		if status == "" {
			status = normalizeLogicalDriveStatus(firstJSONValue(root, "State", "Virtual Drive State"))
		}
		if size == "" {
			size = cleanSize(firstJSONValue(root, "Size"))
		}
		if raidMode == "" {
			raidMode = firstJSONValue(root, "TYPE", "Type", "RAID Type")
		}
		if name == "" {
			name = firstJSONValue(root, "Name")
		}
	}

	if status == "" {
		status = normalizeLogicalDriveStatus(textValue(inputData, "State", "Status of Logical Device"))
	}
	if size == "" {
		size = cleanSize(textValue(inputData, "Size"))
	}
	if raidMode == "" {
		raidMode = textValue(inputData, "TYPE", "Type", "RAID Type")
	}
	if name == "" {
		name = textValue(inputData, "Name")
	}

	data := ReturnData{
		Status:   trim(status),
		Size:     trim(size),
		RaidMode: trim(raidMode),
		Name:     trim(name),
	}

	return append(functions.MarshallJSON(data, indent), "\n"...)
}

// GetPDStatus - get physical drive status
func GetPDStatus(execPath string, controllerID string, deviceID string, indent int) []byte {
	type ReturnData struct {
		Status                 string `json:"status"`
		Model                  string `json:"model"`
		Size                   string `json:"size"`
		CurrentTemperature     string `json:"currenttemperature"`
		Smart                  string `json:"smart"`
		SmartWarn              string `json:"smartwarnings"`
		MediaErrorCount        string `json:"mediaerrorcount"`
		OtherErrorCount        string `json:"othererrorcount"`
		PredictiveFailureCount string `json:"predictivefailurecount"`
	}

	deviceData := strings.Split(deviceID, ":")
	if len(deviceData) != 2 {
		fmt.Printf("Error - wrong device id '%s'.\n", deviceID)
		os.Exit(1)
	}

	inputData := getJSONCommandOutput(execPath, fmt.Sprintf("/c%s/e%s/s%s", controllerID, deviceData[0], deviceData[1]), "show", "all")

	status := ""
	model := ""
	size := ""
	currentTemperature := ""
	smart := ""
	smartWarn := ""
	mediaErrorCount := ""
	otherErrorCount := ""
	predictiveFailureCount := ""

	if root, ok := decodeJSON(inputData); ok {
		if row := findPhysicalDriveRow(root, deviceID); len(row) > 0 {
			status = normalizePhysicalDriveStatus(rowValue(row, "State", "Status"))
			model = rowValue(row, "Model", "Model Number")
			size = cleanSize(rowValue(row, "Size"))
		}

		if status == "" {
			status = normalizePhysicalDriveStatus(firstJSONValue(root, "Firmware state", "State"))
		}
		if model == "" {
			model = firstJSONValue(root, "Model", "Model Number", "Inquiry Data")
		}
		if size == "" {
			size = cleanSize(firstJSONValue(root, "Raw Size", "Coerced Size", "Size"))
		}
		currentTemperature = normalizeTemperature(firstJSONValue(root, "Drive Temperature", "Temperature"))
		smart = normalizeSmartStatus(firstJSONValue(root,
			"S.M.A.R.T alert flagged by drive",
			"SMART alert flagged by drive",
			"S.M.A.R.T Alert",
			"SMART",
		))
		predictiveFailureCount = firstJSONValue(root, "Predictive Failure Count")
		mediaErrorCount = firstJSONValue(root, "Media Error Count")
		otherErrorCount = firstJSONValue(root, "Other Error Count")
	}

	if status == "" {
		status = normalizePhysicalDriveStatus(textValue(inputData, "Firmware state", "State"))
	}
	if model == "" {
		model = textValue(inputData, "Model", "Model Number", "Inquiry Data")
	}
	if size == "" {
		size = cleanSize(textValue(inputData, "Raw Size", "Coerced Size", "Size"))
	}
	if currentTemperature == "" {
		currentTemperature = normalizeTemperature(textValue(inputData, "Drive Temperature", "Temperature"))
	}
	if smart == "" {
		smart = normalizeSmartStatus(textValue(inputData,
			"S.M.A.R.T alert flagged by drive",
			"SMART alert flagged by drive",
			"S.M.A.R.T Alert",
			"SMART",
		))
	}
	if predictiveFailureCount == "" {
		predictiveFailureCount = textValue(inputData, "Predictive Failure Count")
	}
	if mediaErrorCount == "" {
		mediaErrorCount = textValue(inputData, "Media Error Count")
	}
	if otherErrorCount == "" {
		otherErrorCount = textValue(inputData, "Other Error Count")
	}

	if smart == "" {
		smart = "OK"
	}

	smartWarn = normalizeCountStatus(predictiveFailureCount)
	if smart == "OK" && smartWarn != "" && smartWarn != "OK" {
		smart = fmt.Sprintf("Predictive Failure Count is %s", predictiveFailureCount)
	}

	data := ReturnData{
		Status:                 trim(status),
		Model:                  trim(model),
		Size:                   trim(size),
		CurrentTemperature:     trim(currentTemperature),
		Smart:                  trim(smart),
		SmartWarn:              trim(smartWarn),
		MediaErrorCount:        trim(mediaErrorCount),
		OtherErrorCount:        trim(otherErrorCount),
		PredictiveFailureCount: trim(predictiveFailureCount),
	}

	return append(functions.MarshallJSON(data, indent), "\n"...)
}

func getJSONCommandOutput(execPath string, args ...string) []byte {
	args = append(args, "nolog", "J")
	return functions.GetCommandOutput(execPath, args...)
}

func decodeJSON(inputData []byte) (interface{}, bool) {
	var root interface{}
	if err := json.Unmarshal(inputData, &root); err != nil {
		return nil, false
	}
	return root, true
}

func getControllersIDsFromJSON(root interface{}) []string {
	rows := append(findRows(root, "System Overview"), findRows(root, "IT System Overview")...)
	ids := []string{}

	for _, row := range rows {
		if id := rowValue(row, "Ctl", "Controller", "Controller ID"); id != "" {
			ids = append(ids, id)
		}
	}

	if len(ids) == 0 {
		if count := firstJSONValue(root, "Number of Controllers"); count != "" {
			if countInt, err := strconv.Atoi(count); err == nil {
				for i := 0; i < countInt; i++ {
					ids = append(ids, strconv.Itoa(i))
				}
			}
		}
	}

	return sortIDs(uniqueStrings(ids))
}

func getLogicalDrivesIDsFromJSON(root interface{}) []string {
	ids := []string{}
	for _, row := range logicalDriveRows(root) {
		if id := logicalDriveID(row); id != "" {
			ids = append(ids, id)
		}
	}
	return sortIDs(uniqueStrings(ids))
}

func getPhysicalDrivesIDsFromJSON(root interface{}) []string {
	ids := []string{}
	for _, row := range physicalDriveRows(root) {
		if id := physicalDriveID(row); id != "" {
			ids = append(ids, id)
		}
	}
	return sortIDs(uniqueStrings(ids))
}

func controllerModelFromJSON(root interface{}) string {
	for _, row := range findRows(root, "Basics") {
		if model := rowValue(row, "Model", "Product Name", "Controller Model"); model != "" {
			return model
		}
	}

	for _, row := range append(findRows(root, "System Overview"), findRows(root, "IT System Overview")...) {
		if model := rowValue(row, "Model", "Product Name", "Controller Model"); model != "" {
			return model
		}
	}

	return firstJSONValue(root, "Product Name", "Controller Model")
}

func getControllersIDsFromText(inputData []byte) []string {
	if count := textValue(inputData, "Number of Controllers", "Controller Count"); count != "" {
		if countInt, err := strconv.Atoi(count); err == nil {
			ids := []string{}
			for i := 0; i < countInt; i++ {
				ids = append(ids, strconv.Itoa(i))
			}
			return ids
		}
	}

	return sortIDs(uniqueStrings(functions.GetRegexpAllSubmatch(inputData, "(?m)^\\s*(\\d+)\\s+.*(?:MegaRAID|HBA|SAS|PERC|RAID|Tri-Mode|LSI|AVAGO|Broadcom).*")))
}

func getLogicalDrivesIDsFromText(inputData []byte) []string {
	return sortIDs(uniqueStrings(functions.GetRegexpAllSubmatch(inputData, "(?m)^\\s*\\d+/(\\d+)\\s+\\S+.*")))
}

func getPhysicalDrivesIDsFromText(inputData []byte) []string {
	return sortIDs(uniqueStrings(functions.GetRegexpAllSubmatch(inputData, "(?m)^\\s*(\\d+:\\d+)\\s+\\d+\\s+\\S+.*")))
}

func findLogicalDriveRow(root interface{}, deviceID string) map[string]string {
	for _, row := range logicalDriveRows(root) {
		if logicalDriveID(row) == deviceID {
			return row
		}
	}

	rows := logicalDriveRows(root)
	if len(rows) == 1 {
		return rows[0]
	}

	return map[string]string{}
}

func findPhysicalDriveRow(root interface{}, deviceID string) map[string]string {
	for _, row := range physicalDriveRows(root) {
		if physicalDriveID(row) == deviceID {
			return row
		}
	}

	rows := physicalDriveRows(root)
	if len(rows) == 1 {
		return rows[0]
	}

	return map[string]string{}
}

func logicalDriveRows(root interface{}) []map[string]string {
	rows := []map[string]string{}
	for _, key := range []string{"Virtual Drives", "VD LIST", "VD List"} {
		rows = append(rows, findRows(root, key)...)
	}
	return rows
}

func physicalDriveRows(root interface{}) []map[string]string {
	rows := []map[string]string{}
	for _, key := range []string{"Drive Information", "PD LIST", "PD List", "Physical Drives"} {
		rows = append(rows, findRows(root, key)...)
	}
	return rows
}

func logicalDriveID(row map[string]string) string {
	if id := rowValue(row, "VD", "VD ID", "Virtual Drive"); id != "" {
		return id
	}

	if dgVD := rowValue(row, "DG/VD"); dgVD != "" {
		parts := strings.Split(dgVD, "/")
		if len(parts) == 2 {
			return trim(parts[1])
		}
	}

	return ""
}

func physicalDriveID(row map[string]string) string {
	if id := rowValue(row, "EID:Slt", "EID:Slot"); id != "" {
		return id
	}

	enclosure := rowValue(row, "EID", "Enclosure Device ID", "Enclosure")
	slot := rowValue(row, "Slt", "Slot", "Slot Number")
	if enclosure != "" && slot != "" {
		return fmt.Sprintf("%s:%s", enclosure, slot)
	}

	return ""
}

func findRows(root interface{}, wantedKey string) []map[string]string {
	rows := []map[string]string{}
	wanted := canonical(wantedKey)

	var walk func(interface{})
	walk = func(value interface{}) {
		switch v := value.(type) {
		case map[string]interface{}:
			for key, child := range v {
				if canonical(key) == wanted {
					rows = append(rows, rowsFromValue(child)...)
				}
				walk(child)
			}
		case []interface{}:
			for _, child := range v {
				walk(child)
			}
		}
	}

	walk(root)
	return rows
}

func rowsFromValue(value interface{}) []map[string]string {
	rows := []map[string]string{}

	switch v := value.(type) {
	case []interface{}:
		for _, rowValue := range v {
			if row, ok := rowValue.(map[string]interface{}); ok {
				rows = append(rows, stringMap(row))
			}
		}
	case map[string]interface{}:
		rows = append(rows, stringMap(v))
	}

	return rows
}

func stringMap(input map[string]interface{}) map[string]string {
	output := map[string]string{}
	for key, value := range input {
		output[key] = valueToString(value)
	}
	return output
}

func firstJSONValue(root interface{}, keys ...string) string {
	wanted := map[string]bool{}
	for _, key := range keys {
		wanted[canonical(key)] = true
	}

	var walk func(interface{}) string
	walk = func(value interface{}) string {
		switch v := value.(type) {
		case map[string]interface{}:
			for key, child := range v {
				if canonical(key) == "commandstatus" {
					continue
				}
				if wanted[canonical(key)] {
					if data := valueToString(child); data != "" {
						return data
					}
				}
				if data := walk(child); data != "" {
					return data
				}
			}
		case []interface{}:
			for _, child := range v {
				if data := walk(child); data != "" {
					return data
				}
			}
		}

		return ""
	}

	return walk(root)
}

func rowValue(row map[string]string, keys ...string) string {
	for _, wantedKey := range keys {
		wanted := canonical(wantedKey)
		for key, value := range row {
			if canonical(key) == wanted {
				return trim(value)
			}
		}
	}
	return ""
}

func textValue(inputData []byte, keys ...string) string {
	for _, key := range keys {
		re := regexp.MustCompile(`(?mi)^\s*` + regexp.QuoteMeta(key) + `\s*[:=]\s*(.*?)\s*$`)
		if match := re.FindStringSubmatch(string(inputData)); len(match) > 1 {
			return trim(match[1])
		}
	}

	return ""
}

func summarizeControllerStatus(inputData []byte) string {
	healthStatuses := []string{}
	for _, key := range []string{
		"Critical Disks",
		"Failed Disks",
		"Degraded",
		"Offline",
	} {
		if value := textValue(inputData, key); value != "" && !isZeroValue(value) {
			healthStatuses = append(healthStatuses, fmt.Sprintf("%s is %s", key, value))
		}
	}

	if len(healthStatuses) == 0 {
		return "OK"
	}

	return strings.Join(healthStatuses, ", ")
}

func normalizeControllerStatus(status string) string {
	switch canonical(status) {
	case "ok", "optimal", "opt", "optl", "healthy", "success":
		return "OK"
	}
	return trim(status)
}

func normalizeLogicalDriveStatus(status string) string {
	switch canonical(status) {
	case "ok", "optimal", "opt", "optl":
		return "OK"
	}
	return trim(status)
}

func normalizePhysicalDriveStatus(status string) string {
	switch canonical(status) {
	case "ok", "onln", "online", "onlinespunup", "hotspare", "globalhotspare", "dedicatedhotspare", "ghs", "dhs", "jbod":
		return "OK"
	}
	return trim(status)
}

func normalizeBatteryStatus(status string) string {
	switch canonical(status) {
	case "0", "ok", "optimal", "opt", "optl", "healthy", "operational", "ready", "present", "na", "notpresent", "absent":
		return "OK"
	}
	return trim(status)
}

func normalizeCacheStatus(status string) string {
	switch canonical(status) {
	case "0", "ok", "optimal", "opt", "optl", "healthy", "operational", "ready", "no", "none":
		return "OK"
	}

	if strings.Contains(strings.ToLower(status), "not preserved") {
		return "OK"
	}

	return trim(status)
}

func normalizeSmartStatus(status string) string {
	switch canonical(status) {
	case "0", "ok", "no", "false", "none", "notavailable", "na":
		return "OK"
	}
	return trim(status)
}

func normalizeCountStatus(value string) string {
	value = trim(value)
	if value == "" {
		return "OK"
	}
	if isZeroValue(value) {
		return "OK"
	}
	return value
}

func normalizeTemperature(value string) string {
	value = trim(value)
	if value == "" {
		return ""
	}

	re := regexp.MustCompile(`-?\d+`)
	if match := re.FindString(value); match != "" {
		return match
	}

	return value
}

func cleanSize(value string) string {
	value = trim(value)
	if idx := strings.Index(value, "["); idx >= 0 {
		value = trim(value[:idx])
	}
	return value
}

func valueToString(value interface{}) string {
	switch v := value.(type) {
	case string:
		return trim(v)
	case float64:
		if v == float64(int64(v)) {
			return strconv.FormatInt(int64(v), 10)
		}
		return strconv.FormatFloat(v, 'f', -1, 64)
	case bool:
		if v {
			return "true"
		}
		return "false"
	default:
		return ""
	}
}

func trim(value string) string {
	return strings.TrimSpace(value)
}

func canonical(value string) string {
	var builder strings.Builder
	for _, r := range strings.ToLower(value) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			builder.WriteRune(r)
		}
	}
	return builder.String()
}

func uniqueStrings(input []string) []string {
	seen := map[string]bool{}
	output := []string{}

	for _, value := range input {
		value = trim(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		output = append(output, value)
	}

	return output
}

func sortIDs(input []string) []string {
	sort.SliceStable(input, func(i, j int) bool {
		left := idSortKey(input[i])
		right := idSortKey(input[j])

		for idx := 0; idx < len(left) && idx < len(right); idx++ {
			if left[idx] != right[idx] {
				return left[idx] < right[idx]
			}
		}

		return input[i] < input[j]
	})

	return input
}

func idSortKey(value string) []int {
	parts := regexp.MustCompile(`\D+`).Split(value, -1)
	key := []int{}

	for _, part := range parts {
		if part == "" {
			continue
		}

		if value, err := strconv.Atoi(part); err == nil {
			key = append(key, value)
		}
	}

	return key
}

func isZeroValue(value string) bool {
	value = trim(value)
	if value == "" {
		return false
	}

	re := regexp.MustCompile(`^-?\d+`)
	if match := re.FindString(value); match != "" {
		if intValue, err := strconv.Atoi(match); err == nil {
			return intValue == 0
		}
	}

	return false
}
