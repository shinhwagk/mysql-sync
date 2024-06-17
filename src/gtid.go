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

func (g *GtidSets) Query(defaultGtidSetsStr string) (map[string]uint,error) {
	url := fmt.Sprintf("http://%s/file/%s/gtidsets", g.HJDBAddr, g.HJDBDatabase)
	hjdb.Logger.Info("query " + url)
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var hjdbResp HJDBResponse
	if err := json.Unmarshal(body, &hjdbResp) ; err != nil {
		g.Logger.Error(fmt.Sprintf("json unmarshal err: %s", err.Error()))
		return nil ,err

	} else {
		if hjdbResp.State =="err" {
			g.Logger.Error(fmt.Sprintf("hjdb resp err: %s",  hjdbResp.errmsg))
			if hjdbResp.errcode != nil && hjdbResp == "hjdb-001" {
				return make(map[string]uint),nil
			}
			return nil ,fmt.Errorf(hjdbResp.errmsg)
		} else {
			return hjdbResp.Data ,nill
g.Logger.Debug(fmt.Sprintf("Persist gtidsets map '%v' success.",gssm) )
		}
	}
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
		g.Logger.Error(fmt.Sprintf("read body err: %s",   err.Error()))
	}

	var hjdbResp HJDBResponse
	if err := json.Unmarshal(body, &hjdbResp) ; err != nil {
		g.Logger.Error(fmt.Sprintf("json unmarshal err: %s", err.Error()))

	} else {
		if hjdbResp.State =="err" {
			g.Logger.Error(fmt.Sprintf("hjdb resp err: %s",  hjdbResp.errmsg))
		} else {
g.Logger.Debug(fmt.Sprintf("Persist gtidsets map '%v' success.",gssm) )
		}
	}
}