package reports

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"sort"
	"strconv"

	"encoding/base64"

	jira "github.com/andygrunwald/go-jira/v2/cloud"
	"gopkg.in/yaml.v3"

	"image/jpeg"

	"strings"

	"github.com/tdewolff/canvas"
	"github.com/tdewolff/canvas/renderers/rasterizer"
	_ "github.com/tdewolff/canvas/renderers/svg"
	"github.com/xo/echartsgoja"
)

const (
	maxIssuesRetrieved = 500

	gaugeWidth  = 100
	gaugeHeight = 100

	barWidth  = 400
	barHeight = 100

	dpi         = 200.0
	jpegQuality = 85

	greenColor  = "#00FF00"
	yellowColor = "#E6B800"
	blueColor   = "#015CE6"

	yes  = "yes"
	no   = "no"
	both = "both"
)

type patTransport struct {
	Token string
}
type jiraColor struct {
	Disabled bool   `json:"disabled"`
	ID       string `json:"id"`
	Self     string `json:"self"`
	Value    string `json:"value"`
}

type jiraState struct {
	Fields struct {
		Issuetype struct {
			AvatarID    int    `json:"avatarId"`
			Description string `json:"description"`
			IconURL     string `json:"iconUrl"`
			ID          string `json:"id"`
			Name        string `json:"name"`
			Self        string `json:"self"`
			Subtask     bool   `json:"subtask"`
		} `json:"issuetype"`
		Priority struct {
			IconURL string `json:"iconUrl"`
			ID      string `json:"id"`
			Name    string `json:"name"`
			Self    string `json:"self"`
		} `json:"priority"`
		Status struct {
			Description    string `json:"description"`
			IconURL        string `json:"iconUrl"`
			ID             string `json:"id"`
			Name           string `json:"name"`
			Self           string `json:"self"`
			StatusCategory struct {
				ColorName string `json:"colorName"`
				ID        int    `json:"id"`
				Key       string `json:"key"`
				Name      string `json:"name"`
				Self      string `json:"self"`
			} `json:"statusCategory"`
		} `json:"status"`
		Summary string `json:"summary"`
	} `json:"fields"`
	ID   string `json:"id"`
	Key  string `json:"key"`
	Self string `json:"self"`
}

type JiraFilter struct {
	Name      string `yaml:"name"`
	URL       string `yaml:"url"`
	Variables int    `yaml:"variables"`
	Filter    string `yaml:"filter"`
}

//go:embed filters/bugstatus.yml
var bugStatusFiltersYAML []byte

type stats struct {
	colorGreen    int
	colorRed      int
	colorYellow   int
	colorNoStatus int
	colorTotal    int

	statusClosed         int
	statusReleasePending int
	statusNew            int
	statusToDo           int
	statusInProgress     int
	statusDevComplete    int
	statusPlaning        int
}

func (t *patTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("Authorization", "Bearer "+t.Token)
	return http.DefaultTransport.RoundTrip(req)
}

func getFilterFromRelease(release, customerFacing string) string {
	switch customerFacing {
	case yes:
		return `(project = "Cloud-native Network Functions" and issuetype = epic or project = "OpenShift Edge Enablement" and "Portfolio Solutions" = Telco or project = "KNI QE - System Test" and "Portfolio Solutions" = Telco and issuetype = epic and status not in (Obsolete, "Won't Fix / Obsolete", "Won't Do", "WON'T FIX", "Won't Fix / Duplicate", WONTFIX) or issue = OCPNODE-2305) and issuetype = epic and fixVersion = openshift-` + release + ` and Planning = "Customer Facing"` //nolint:lll
	case no:
		return `(project = "Cloud-native Network Functions" and issuetype = epic or project = "OpenShift Edge Enablement" and "Portfolio Solutions" = Telco or project = "KNI QE - System Test" and "Portfolio Solutions" = Telco and issuetype = epic and status not in (Obsolete, "Won't Fix / Obsolete", "Won't Do", "WON'T FIX", "Won't Fix / Duplicate", WONTFIX) or issue = OCPNODE-2305) and issuetype = epic and fixVersion = openshift-` + release + ` and (Planning != "Customer Facing" or Planning is EMPTY)` //nolint:lll
	case both:
		return `(project = "Cloud-native Network Functions" and issuetype = epic or project = "OpenShift Edge Enablement" and "Portfolio Solutions" = Telco or project = "KNI QE - System Test" and "Portfolio Solutions" = Telco and issuetype = epic and status not in (Obsolete, "Won't Fix / Obsolete", "Won't Do", "WON'T FIX", "Won't Fix / Duplicate", WONTFIX) or issue = OCPNODE-2305) and issuetype = epic and fixVersion = openshift-` + release //nolint:lll
	default:
		log.Fatalf("customerFacing argument: %s not supported. Use %s, %s, or %s", customerFacing, yes, no, both)
	}
	return ""
}

func GetMarkdownReport(jiraURL, personalAccessToken, filterQuery, release, customerFacing string) { //nolint:funlen
	httpClient := &http.Client{
		Transport: &patTransport{Token: personalAccessToken},
	}

	client, err := jira.NewClient(jiraURL, httpClient)
	if err != nil {
		log.Fatal(err)
	}

	statistics := stats{}
	outputYellow := ""
	outputRed := ""
	outputGreen := ""
	outputNone := ""
	jiraOption := jira.SearchOptions{MaxResults: maxIssuesRetrieved}

	if filterQuery == "" {
		filterQuery = getFilterFromRelease(release, customerFacing)
	}

	var issues []jira.Issue
	issues, _, err = client.Issue.Search(context.TODO(), filterQuery, &jiraOption)
	if err != nil {
		log.Fatal(err)
	}
	for _, issue := range issues {
		color := getCustomField("customfield_12320845", issue)
		statusSummary := getCustomField("customfield_12320841", issue)
		state := issue.Fields.Status.Name

		output := fmt.Sprintf("  - [%s: %s](https://issues.redhat.com/browse/%s)\n",
			issue.Key, issue.Fields.Summary, issue.Key)

		nbsp := "\u00A0" // Non-breaking space
		re := regexp.MustCompile(`[\s\t\n\r` + regexp.QuoteMeta(nbsp) + `]+`)
		cleaned := re.ReplaceAllString(statusSummary, "")

		// Add bullet
		re = regexp.MustCompile(`[\n]+`)
		statusSummaryBullets := re.ReplaceAllString(statusSummary, "\n    - ")

		// Remove empty lines
		re = regexp.MustCompile(`(?m)^[\s` + regexp.QuoteMeta(nbsp) + `\-]*$`)
		statusSummaryBullets = re.ReplaceAllString(statusSummaryBullets, "")

		if cleaned != "" {
			output += fmt.Sprintf("    - %s\n", statusSummaryBullets)
		}

		switch color {
		case "Green":
			statistics.colorGreen++
			outputGreen += output
		case "Yellow":
			statistics.colorYellow++
			outputYellow += output
		case "Red":
			statistics.colorRed++
			outputRed += output
		default:
			statistics.colorNoStatus++
			outputNone += output
		}

		switch state {
		case "Closed":
			statistics.statusClosed++
		case "Release Pending":
			statistics.statusReleasePending++
		case "Planning":
			statistics.statusPlaning++
		case "To Do":
			statistics.statusToDo++
		case "In Progress":
			statistics.statusInProgress++
		case "Dev Complete":
			statistics.statusDevComplete++
		case "New":
			statistics.statusNew++
		default:
			statistics.statusNew++
		}
	}

	statistics.colorTotal = statistics.colorGreen + statistics.colorRed + statistics.colorYellow + statistics.colorNoStatus
	greenGauge := generateGaugeDataURI(gaugeWidth, gaugeHeight,
		"green", greenColor, statistics.colorGreen, statistics.colorTotal)
	redGauge := generateGaugeDataURI(gaugeWidth, gaugeHeight,
		"red", "red", statistics.colorRed, statistics.colorTotal)
	yellowGauge := generateGaugeDataURI(gaugeWidth, gaugeHeight,
		"yellow", yellowColor, statistics.colorYellow, statistics.colorTotal)
	noStatusGauge := generateGaugeDataURI(gaugeWidth, gaugeHeight,
		"no status", "grey", statistics.colorNoStatus, statistics.colorTotal)

	fmt.Println("\n\n" + redGauge + yellowGauge + greenGauge + noStatusGauge)

	labels := []string{"CLOSED", "RELEASE PENDING", "IN PROGRESS", "DEV COMPLETE", "PLANNING", "TO DO", "NEW"}
	values := []int{statistics.statusClosed,
		statistics.statusReleasePending,
		statistics.statusInProgress,
		statistics.statusDevComplete,
		statistics.statusPlaning,
		statistics.statusToDo,
		statistics.statusNew,
	}
	bar := generateBarDataURI(barWidth, barHeight, labels, values)
	fmt.Println(bar)

	fmt.Printf("<br>\n\n<span style=\"background-color:red; color:white\">RED</span>\n%s\n"+
		"<span style=\"background-color:yellow; color:black\">YELLOW</span>\n%s\n"+
		"<span style=\"background-color:grey; color:white\">NO STATUS</span>\n%s", outputRed, outputYellow, outputNone)
}

func GetBugStatusReport(jiraURL, personalAccessToken, releaseCutoffDate, fromDate string) {
	filters, err := loadFilters(bugStatusFiltersYAML)
	if err != nil {
		log.Fatalf("Cannot load embedded filters, err:%v", err)
	}
	for _, filter := range filters {
		allVariables := []string{fromDate, releaseCutoffDate, releaseCutoffDate}
		patchedFilter := fmt.Sprintf(filter.Filter,toAnySliceNFirst(allVariables, filter.Variables)...)

		bar := getBugStatusDiagram(jiraURL, personalAccessToken, patchedFilter)
		println(bar)
	}
}
func toAnySliceNFirst(slice []string, n int) []any {
    result := make([]any, len(slice))
    for i, v := range slice {
        result[i] = v
    }
	if n>len(slice){
		n=len(slice)
	}
    return result[:n]
}
func getBugStatusDiagram(jiraURL, personalAccessToken, filterQuery string) string {
	jiraOption := jira.SearchOptions{MaxResults: maxIssuesRetrieved}
	httpClient := &http.Client{
		Transport: &patTransport{Token: personalAccessToken},
	}

	client, err := jira.NewClient(jiraURL, httpClient)
	if err != nil {
		log.Fatal(err)
	}

	var issues []jira.Issue
	issues, _, err = client.Issue.Search(context.TODO(), filterQuery, &jiraOption)
	if err != nil {
		log.Fatal(err)
	}
	componentsMap := map[string]int{}

	for _, issue := range issues {
		components := issue.Fields.Components
		for _, component := range components {
			componentsMap[component.Name]++
		}
	}

	keys, values := getKeyValueFromMap(componentsMap)
	return generateBarDataURI(barWidth, barHeight, keys, values)
}

func getKeyValueFromMap(aMap map[string]int) (keys []string, values []int) {
	type keyValue struct {
		Key   string
		Value int
	}

	var keyValueSlice []keyValue
	for k, v := range aMap {
		keyValueSlice = append(keyValueSlice, keyValue{k, v})
	}

	// Sort
	sort.Slice(keyValueSlice, func(i, j int) bool {
		return keyValueSlice[i].Value < keyValueSlice[j].Value
	})

	for _, item := range keyValueSlice {
		keys = append(keys, item.Key)
		values = append(values, item.Value)
	}

	return keys, values
}

func loadFilters(filterString []byte) (filters []JiraFilter, err error) {
	err = yaml.Unmarshal(filterString, &filters)
	if err != nil {
		return nil, err
	}
	return filters, nil
}

func getCustomField(name string, issue jira.Issue) string {
	if value, ok := issue.Fields.Unknowns[name]; ok {
		str, ok := value.(string)

		if ok {
			return str
		}
		switch name {
		case "customfield_12320845":
			aJson, err := json.MarshalIndent(value, "", "  ")
			if err != nil {
				return ""
			}
			object := jiraColor{}
			err = json.Unmarshal(aJson, &object)
			if err != nil {
				return ""
			}
			return object.Value
		case "customfield_12318341":
			aJson, err := json.MarshalIndent(value, "", "  ")
			if err != nil {
				return ""
			}
			object := jiraState{}
			err = json.Unmarshal(aJson, &object)
			if err != nil {
				return ""
			}
			return object.Fields.Status.Name
		}
	}
	return ""
}

func sVGStringToPNGDataURI(svgSrc string, quality int, width, height int) (string, error) {
	// Remove style because of parsing error
	styleRe := regexp.MustCompile(`(?is)<style.*?>.*?</style>`)
	cleaned := styleRe.ReplaceAllString(svgSrc, "")

	r := strings.NewReader(cleaned)
	c, err := canvas.ParseSVG(r)
	if err != nil {
		return "", fmt.Errorf("parse SVG: %w", err)
	}

	img := rasterizer.Draw(c, canvas.DPI(dpi), canvas.DefaultColorSpace)

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: quality}); err != nil {
		return "", fmt.Errorf("JPEG encoding: %w", err)
	}

	b64 := base64.StdEncoding.EncodeToString(buf.Bytes())

	return fmt.Sprintf("\n\n<img src=\"data:image/jpeg;base64,%s\" width=\"%d\" height=\"%d\">", b64, width, height), nil
}

func generateBarDataURI(width, height int, labels []string, values []int) string {
	total := 0
	valueStrings := []string{}
	for _, v := range values {
		total += v
		valueStrings = append(valueStrings, strconv.Itoa(v))
	}
	valueStrings = append([]string{strconv.Itoa(total)}, valueStrings...)

	percentagesStrings := []string{}
	for _, v := range values {
		percentagesStrings = append(percentagesStrings, strconv.FormatFloat(float64(v)/float64(total)*100.0, 'f', 2, 64))
	}
	percentagesStrings = append([]string{"100"}, percentagesStrings...)

	labels = append([]string{"TOTAL"}, labels...)

	for i := range labels {
		labels[i] = fmt.Sprintf("%q", labels[i])
	}
	renderedValues := []string{}
	renderedPercentages := []string{}
	renderedLabels := []string{}
	for i := range valueStrings {
		if valueStrings[i] != "0" {
			renderedValues = append(renderedValues, valueStrings[i])
			renderedPercentages = append(renderedPercentages, percentagesStrings[i])
			renderedLabels = append(renderedLabels, labels[i])
		}
	}

	patchedOptions := fmt.Sprintf(simpleOptsBar,
		fmt.Sprintf("[%s]", strings.Join(renderedLabels, ", ")),
		fmt.Sprintf("[%s]", strings.Join(renderedValues, ", ")),
		fmt.Sprintf("[%s]", strings.Join(renderedPercentages, ", ")),
		blueColor)

	echarts := echartsgoja.New(echartsgoja.WithWidthHeight(width, height))
	svg, err := echarts.RenderOptions(context.Background(), patchedOptions)
	if err != nil {
		log.Fatal(err)
	}

	dataURI, err := sVGStringToPNGDataURI(svg, jpegQuality, width, height)
	if err != nil {
		log.Fatalf("conversion failed: %v", err)
	}
	return dataURI
}

const simpleOptsBar = `{
  "backgroundColor": "white",
  "tooltip": {
    "trigger": "axis"
  },
  "grid": {
    "left": 160 ,
       "top": 10, 
    "bottom": 10, 
    "right": 50  
  },
  "xAxis": {
    "type": "value",
    "max": 100,
    "axisLine": {
      "show": false
    },
    "axisTick": {
      "show": false  
    },
    "splitLine": {
      "show": false
    },
    "axisLabel": {
  "show": false
}
  },
  "yAxis": {
    "type": "category",
    "data": %s,
    "axisLabel": {
      "align": "left",
      "margin": 150 ,
      "fontSize": 10 
    },
    "axisLine": {
      "show": false
    },
    "axisTick": {
      "show": false  
    }
  },
  "series": [
    {
      "name": "Issue Count",
      "type": "bar",
      "data": %s,
      "itemStyle": {
        "color": "transparent"
      },
      "label": {
        "show": true,
        "align": "center",         
        "formatter": "{c}",
        "position": "left",
        "color": "#000000",
          "fontSize": 10,
          "verticalAlign": "middle",
          "offset": [-20, 4],  
          "padding": [5, 5, 5, 5]
      },
      
      "barWidth": "70%%",
    "barGap": "-100%%",
    "barCategoryGap": "-50%%"
    },
        {
      "name": "Issue percent",
      "type": "bar",
      "data": %s,
      "itemStyle": {
        "color": "%s"
      },
      "label": {
        "show": true,
        "align": "center",         
        "formatter": "{c}%%",
        "position": "right",
        "color": "#000000",
          "fontSize": 10,
          "verticalAlign": "middle",
          "offset": [20, 4],  
          "padding": [5, 5, 5, 5]
      },
      
      "barWidth": "70%%",
    "barGap": "-100%%",
    "barCategoryGap": "-50%%"
    }
  ]
}
`

func generateGaugeDataURI(width, height int, label, color string, value, total int) string { //nolint:unparam
	percent := float64(value) / float64(total) * 100.0 //nolint:mnd
	patchedOptions := fmt.Sprintf(simpleOptsGauge, color, color, int(percent), strings.ToUpper(label), value, total)
	echarts := echartsgoja.New(echartsgoja.WithWidthHeight(width, height))
	svg, err := echarts.RenderOptions(context.Background(), patchedOptions)
	if err != nil {
		log.Fatal(err)
	}
	dataURI, err := sVGStringToPNGDataURI(svg, jpegQuality, width, height)
	if err != nil {
		log.Fatalf("conversion failed: %v", err)
	}
	return dataURI
}

const simpleOptsGauge = `{
  "backgroundColor": "white",
  "series": [
    {
      "name": "value",
      "type": "gauge",
      "startAngle": 180,
      "endAngle": 0,
      "center": ["50%%", "75%%"],
      "radius": "100%%",
      "min": 0,
      "max": 100,
      "pointer": { "show": false },
      "axisLine": {
        "lineStyle": {
          "width": 10,
          "color": [[1, "#D3D3D3"]],
          "cap": "butt"
        },
        "roundCap": true
      },
      "progress": {
        "show": true,
        "width": 10,
        "roundCap": true,
        "itemStyle": {
          "color": "%s"
        }
      },
      "splitLine": { "show": false },
      "axisTick": { "show": false },
      "axisLabel": { "show": false },
      "detail": {
        "backgroundColor": "transparent",
        "formatter": "{value}%%",
        "valueAnimation": true,
        "offsetCenter": [0, "-20%%"],
        "fontSize": 10
      },
      "title": {
        "offsetCenter": [0, "20%%"],
        "fontSize": 10	,
        "color": "%s"
      },
      "data": [
        {
          "value": %d,
          "name": "%s"
        }
      ]
    },
    {
          "name": "start",
      "type": "gauge",
      "startAngle": 180,
      "endAngle": 0,
      "center": ["50%%", "75%%"],
      "radius": "100%%",
      "min": 0,
      "max": 100,
      "pointer": { "show": false },
      "axisLine": { "show": false },
      "progress": { "show": false },
      "splitLine": { "show": false },
      "axisTick": { "show": false },
      "axisLabel": { "show": false },
      "detail": {
        "formatter": "{value}",
        "fontSize": 10,
        "offsetCenter": ["-85%%","40%%"],
        "backgroundColor":"transparent"
      },
      "title": {
        "offsetCenter": ["-120%%", "0%%"],
		"fontSize": 0
      },
      "data": [
        {
          "value": %d,
          "name": "0"
        }
      ]
    },
    {
      "name": "end",
      "type": "gauge",
      "startAngle": 180,
      "endAngle": 0,
      "center": ["50%%", "75%%"],
      "radius": "100%%",
      "min": 0,
      "max": 100,
      "pointer": { "show": false },
      "axisLine": { "show": false },
      "progress": { "show": false },
      "splitLine": { "show": false },
      "axisTick": { "show": false },
      "axisLabel": { "show": false },
      "detail": {
        "formatter": "{value}",
        "fontSize": 10,
        "offsetCenter": ["85%%","40%%"],
        "backgroundColor":"transparent"
      },
      "title": {
        "offsetCenter": ["-120%%", "0%%"],
		"fontSize": 0
      },
      "data": [
        {
          "value": %d,
          "name": "0"
        }
      ]
    }
  ]
}
`
