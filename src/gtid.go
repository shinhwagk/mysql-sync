type HJDBResponse struct {
	State   *string      `json:"state"`
	Data    *map[string]uint `json:"data"`
	DB      *string      `json:"db"`
	Tab     *string      `json:"tab`
	ErrMsg  *string      `json:"errmsg"`
	ErrCode *string      `json:"errcode"`
	Store   *string      `json.store`
}

type GtidSets struct {
 HJDBAddr string
 HJDBDatabase  string

 Logger *Logger
}

func (g *GtidSets) Query() {

}

// gtid sets map gssm
func (g *GtidSets) Persist(gssm map[string]uint) {
	jsonData, err := json.Marshal(gssm)
	if err != nil {
		g.Logger.Error(fmt.Sprintf("json marshal err: %s", err))
	}

	url := fmt.Sprintf("http://%s/file/%s/gtidsets", g.HJDBAddr, hjdb.DB)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		g.Logger.Error(fmt.Sprintf("request '%s' err: %s",url, err.Error()))
	}
	defer resp.Body.Close()

	body, err = io.ReadAll(resp.Body)
	if err != nil {
		hjdb.Logger.Error(fmt.Sprintf("read body err: %s",   err.Error()))
	}

	var hjdbResp HJDBResponse
	if err := json.Unmarshal(body, &hjdbResp) ; err != nil {
		hjdb.Logger.Error(fmt.Sprintf("json unmarshal err: %s", err.Error()))

	} else {
		if hjdbResp.State =="err" {
			g.Logger.Error(fmt.Sprintf("hjdb resp err: %s",  hjdbResp.errmsg))
		} else {
g.Logger.Debug("Persist gtidsets map success.")
		}
	}
}