package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"sort"
	"strings"
	"text/tabwriter"
	"time"
)

const (
	defaultBoardID = "XobnRsYv"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

var errHelpDisplayed = errors.New("help displayed")

type Config struct {
	APIKey  string
	Token   string
	BoardID string
	JSON    bool
}

type Client struct {
	BaseURL string
	APIKey  string
	Token   string
	HTTP    *http.Client
}

type trelloError struct {
	Message string `json:"message"`
	Error   string `json:"error"`
}

type Board struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	URL    string `json:"url"`
	Closed bool   `json:"closed"`
}

type TrelloList struct {
	ID     string  `json:"id"`
	Name   string  `json:"name"`
	Closed bool    `json:"closed"`
	Pos    float64 `json:"pos"`
}

type Card struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Desc     string `json:"desc"`
	IDList   string `json:"idList"`
	ShortURL string `json:"shortUrl"`
	URL      string `json:"url"`
	Due      string `json:"due"`
	Closed   bool   `json:"closed"`
}

type CommentAction struct {
	ID   string `json:"id"`
	Type string `json:"type"`
	Date string `json:"date"`
	Data struct {
		Text string `json:"text"`
	} `json:"data"`
	MemberCreator struct {
		Username string `json:"username"`
		FullName string `json:"fullName"`
	} `json:"memberCreator"`
}

type Checklist struct {
	ID         string          `json:"id"`
	Name       string          `json:"name"`
	CheckItems []ChecklistItem `json:"checkItems"`
}

type ChecklistItem struct {
	ID    string  `json:"id"`
	Name  string  `json:"name"`
	State string  `json:"state"`
	Pos   float64 `json:"pos"`
}

func main() {
	cfg, args, help, err := parseGlobal(os.Args[1:])
	if err != nil {
		fatalf("%v\n\n", err)
	}

	if help {
		if len(args) == 0 {
			printRootHelp()
			return
		}
		printCommandHelp(args[0])
		return
	}

	if len(args) == 0 {
		printRootHelp()
		return
	}

	cmd := args[0]
	if cmd == "help" {
		if len(args) > 1 {
			printCommandHelp(args[1])
			return
		}
		printRootHelp()
		return
	}
	if cmd == "version" {
		fmt.Printf("trelli %s (commit %s, built %s)\n", version, commit, date)
		return
	}

	remaining := args[1:]
	var client *Client
	if !shouldSkipAuthForHelp(remaining) {
		client, err = newClient(cfg)
		if err != nil {
			fatalf("%v\n", err)
		}
	}

	switch cmd {
	case "boards":
		err = runBoards(client, cfg, remaining)
	case "lists":
		err = runLists(client, cfg, remaining)
	case "cards":
		err = runCards(client, cfg, remaining)
	case "comments":
		err = runComments(client, cfg, remaining)
	case "checklists":
		err = runChecklists(client, cfg, remaining)
	default:
		err = fmt.Errorf("unknown command %q", cmd)
	}

	if err != nil {
		if errors.Is(err, errHelpDisplayed) {
			return
		}
		fatalf("%v\n", err)
	}
}

func parseGlobal(args []string) (Config, []string, bool, error) {
	cfg := Config{
		APIKey:  strings.TrimSpace(os.Getenv("TRELLO_API_KEY")),
		Token:   strings.TrimSpace(os.Getenv("TRELLO_TOKEN")),
		BoardID: strings.TrimSpace(os.Getenv("TRELLO_BOARD_ID")),
	}
	if cfg.BoardID == "" {
		cfg.BoardID = defaultBoardID
	}

	fs := flag.NewFlagSet("trelli", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var help bool
	fs.StringVar(&cfg.APIKey, "key", cfg.APIKey, "Trello API key (default: TRELLO_API_KEY)")
	fs.StringVar(&cfg.Token, "token", cfg.Token, "Trello token (default: TRELLO_TOKEN)")
	fs.StringVar(&cfg.BoardID, "board", cfg.BoardID, "Default board id or shortLink (default: TRELLO_BOARD_ID or XobnRsYv)")
	fs.BoolVar(&cfg.JSON, "json", false, "Print raw JSON")
	fs.BoolVar(&help, "h", false, "Show help")
	fs.BoolVar(&help, "help", false, "Show help")

	if err := fs.Parse(args); err != nil {
		return Config{}, nil, false, err
	}

	return cfg, fs.Args(), help, nil
}

func newClient(cfg Config) (*Client, error) {
	if cfg.APIKey == "" || cfg.Token == "" {
		return nil, errors.New("missing credentials: set TRELLO_API_KEY and TRELLO_TOKEN (or pass --key/--token)")
	}
	return &Client{
		BaseURL: "https://api.trello.com",
		APIKey:  cfg.APIKey,
		Token:   cfg.Token,
		HTTP: &http.Client{
			Timeout: 20 * time.Second,
		},
	}, nil
}

func (c *Client) do(method, p string, query, form url.Values, out any) error {
	if query == nil {
		query = make(url.Values)
	}
	query.Set("key", c.APIKey)
	query.Set("token", c.Token)

	u, err := url.Parse(c.BaseURL)
	if err != nil {
		return err
	}
	u.Path = path.Join(u.Path, p)
	u.RawQuery = query.Encode()

	var body io.Reader
	if method != http.MethodGet && form != nil {
		body = strings.NewReader(form.Encode())
	}

	req, err := http.NewRequest(method, u.String(), body)
	if err != nil {
		return err
	}
	if method != http.MethodGet && form != nil {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		var apiErr trelloError
		_ = json.NewDecoder(resp.Body).Decode(&apiErr)
		if apiErr.Message != "" {
			return fmt.Errorf("trello API error (%d): %s", resp.StatusCode, apiErr.Message)
		}
		if apiErr.Error != "" {
			return fmt.Errorf("trello API error (%d): %s", resp.StatusCode, apiErr.Error)
		}
		raw, _ := io.ReadAll(resp.Body)
		if len(strings.TrimSpace(string(raw))) > 0 {
			return fmt.Errorf("trello API error (%d): %s", resp.StatusCode, strings.TrimSpace(string(raw)))
		}
		return fmt.Errorf("trello API error (%d)", resp.StatusCode)
	}

	if out == nil {
		return nil
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		if errors.Is(err, io.EOF) {
			return nil
		}
		return err
	}
	return nil
}

func runBoards(client *Client, cfg Config, args []string) error {
	if len(args) == 0 {
		printBoardsHelp()
		return nil
	}

	switch args[0] {
	case "-h", "--help", "help":
		printBoardsHelp()
		return nil
	case "list":
		fs := flag.NewFlagSet("boards list", flag.ContinueOnError)
		fs.SetOutput(io.Discard)
		var filter string
		fs.StringVar(&filter, "filter", "", "Case-insensitive substring filter on board name")
		if err := parseFlagSet(fs, args[1:], printBoardsHelp); err != nil {
			return err
		}

		query := url.Values{}
		query.Set("fields", "id,name,url,closed")
		var boards []Board
		if err := client.do(http.MethodGet, "/1/members/me/boards", query, nil, &boards); err != nil {
			return err
		}

		if filter != "" {
			needle := strings.ToLower(strings.TrimSpace(filter))
			filtered := make([]Board, 0, len(boards))
			for _, b := range boards {
				if strings.Contains(strings.ToLower(b.Name), needle) {
					filtered = append(filtered, b)
				}
			}
			boards = filtered
		}

		sort.Slice(boards, func(i, j int) bool { return boards[i].Name < boards[j].Name })
		if cfg.JSON {
			return printJSON(boards)
		}
		return printBoardsTable(boards)
	default:
		return fmt.Errorf("unknown boards subcommand %q", args[0])
	}
}

func runLists(client *Client, cfg Config, args []string) error {
	if len(args) == 0 {
		printListsHelp()
		return nil
	}

	switch args[0] {
	case "-h", "--help", "help":
		printListsHelp()
		return nil
	case "list":
		fs := flag.NewFlagSet("lists list", flag.ContinueOnError)
		fs.SetOutput(io.Discard)
		boardID := cfg.BoardID
		fs.StringVar(&boardID, "board", boardID, "Board id or shortLink")
		if err := parseFlagSet(fs, args[1:], printListsHelp); err != nil {
			return err
		}
		if strings.TrimSpace(boardID) == "" {
			return errors.New("missing --board and no default board configured")
		}

		lists, err := fetchBoardLists(client, boardID)
		if err != nil {
			return err
		}
		sort.Slice(lists, func(i, j int) bool { return lists[i].Pos < lists[j].Pos })
		if cfg.JSON {
			return printJSON(lists)
		}
		return printListsTable(lists)
	default:
		return fmt.Errorf("unknown lists subcommand %q", args[0])
	}
}

func runCards(client *Client, cfg Config, args []string) error {
	if len(args) == 0 {
		printCardsHelp()
		return nil
	}

	switch args[0] {
	case "-h", "--help", "help":
		printCardsHelp()
		return nil
	case "list":
		fs := flag.NewFlagSet("cards list", flag.ContinueOnError)
		fs.SetOutput(io.Discard)
		var listID, listName string
		boardID := cfg.BoardID
		limit := 100
		fs.StringVar(&listID, "list", "", "List id")
		fs.StringVar(&listName, "list-name", "", "List name (resolved on board)")
		fs.StringVar(&boardID, "board", boardID, "Board id or shortLink (used with --list-name)")
		fs.IntVar(&limit, "limit", limit, "Max cards to return")
		if err := parseFlagSet(fs, args[1:], printCardsHelp); err != nil {
			return err
		}
		resolvedListID, err := resolveListID(client, boardID, listID, listName)
		if err != nil {
			return err
		}

		query := url.Values{}
		query.Set("fields", "id,name,desc,idList,shortUrl,url,due,closed")
		query.Set("limit", fmt.Sprintf("%d", limit))
		var cards []Card
		if err := client.do(http.MethodGet, "/1/lists/"+url.PathEscape(resolvedListID)+"/cards", query, nil, &cards); err != nil {
			return err
		}
		if cfg.JSON {
			return printJSON(cards)
		}
		return printCardsTable(cards)

	case "show":
		fs := flag.NewFlagSet("cards show", flag.ContinueOnError)
		fs.SetOutput(io.Discard)
		var cardID string
		fs.StringVar(&cardID, "card", "", "Card id")
		if err := parseFlagSet(fs, args[1:], printCardsHelp); err != nil {
			return err
		}
		if strings.TrimSpace(cardID) == "" {
			return errors.New("cards show requires --card")
		}

		query := url.Values{}
		query.Set("fields", "id,name,desc,idList,shortUrl,url,due,closed")
		var card Card
		if err := client.do(http.MethodGet, "/1/cards/"+url.PathEscape(cardID), query, nil, &card); err != nil {
			return err
		}
		if cfg.JSON {
			return printJSON(card)
		}
		return printCardsTable([]Card{card})

	case "create":
		fs := flag.NewFlagSet("cards create", flag.ContinueOnError)
		fs.SetOutput(io.Discard)
		var listID, listName, name, desc, due, labels, members string
		boardID := cfg.BoardID
		fs.StringVar(&listID, "list", "", "List id")
		fs.StringVar(&listName, "list-name", "", "List name (resolved on board)")
		fs.StringVar(&boardID, "board", boardID, "Board id or shortLink (used with --list-name)")
		fs.StringVar(&name, "name", "", "Card title")
		fs.StringVar(&desc, "desc", "", "Card description")
		fs.StringVar(&due, "due", "", "Due date/time (ISO-8601)")
		fs.StringVar(&labels, "labels", "", "Comma-separated Trello label IDs")
		fs.StringVar(&members, "members", "", "Comma-separated member IDs")
		if err := parseFlagSet(fs, args[1:], printCardsHelp); err != nil {
			return err
		}
		if strings.TrimSpace(name) == "" {
			return errors.New("cards create requires --name")
		}
		resolvedListID, err := resolveListID(client, boardID, listID, listName)
		if err != nil {
			return err
		}

		form := url.Values{}
		form.Set("idList", resolvedListID)
		form.Set("name", name)
		if strings.TrimSpace(desc) != "" {
			form.Set("desc", desc)
		}
		if strings.TrimSpace(due) != "" {
			form.Set("due", due)
		}
		if strings.TrimSpace(labels) != "" {
			form.Set("idLabels", labels)
		}
		if strings.TrimSpace(members) != "" {
			form.Set("idMembers", members)
		}

		var card Card
		if err := client.do(http.MethodPost, "/1/cards", nil, form, &card); err != nil {
			return err
		}
		if cfg.JSON {
			return printJSON(card)
		}
		return printCardsTable([]Card{card})

	case "move":
		fs := flag.NewFlagSet("cards move", flag.ContinueOnError)
		fs.SetOutput(io.Discard)
		var cardID, listID, listName string
		boardID := cfg.BoardID
		fs.StringVar(&cardID, "card", "", "Card id")
		fs.StringVar(&listID, "list", "", "Destination list id")
		fs.StringVar(&listName, "list-name", "", "Destination list name (resolved on board)")
		fs.StringVar(&boardID, "board", boardID, "Board id or shortLink (used with --list-name)")
		if err := parseFlagSet(fs, args[1:], printCardsHelp); err != nil {
			return err
		}
		if strings.TrimSpace(cardID) == "" {
			return errors.New("cards move requires --card")
		}
		resolvedListID, err := resolveListID(client, boardID, listID, listName)
		if err != nil {
			return err
		}

		form := url.Values{}
		form.Set("idList", resolvedListID)
		var card Card
		if err := client.do(http.MethodPut, "/1/cards/"+url.PathEscape(cardID), nil, form, &card); err != nil {
			return err
		}
		if cfg.JSON {
			return printJSON(card)
		}
		return printCardsTable([]Card{card})

	case "archive":
		fs := flag.NewFlagSet("cards archive", flag.ContinueOnError)
		fs.SetOutput(io.Discard)
		var cardID string
		fs.StringVar(&cardID, "card", "", "Card id")
		if err := parseFlagSet(fs, args[1:], printCardsHelp); err != nil {
			return err
		}
		if strings.TrimSpace(cardID) == "" {
			return errors.New("cards archive requires --card")
		}

		form := url.Values{}
		form.Set("closed", "true")
		var card Card
		if err := client.do(http.MethodPut, "/1/cards/"+url.PathEscape(cardID), nil, form, &card); err != nil {
			return err
		}
		if cfg.JSON {
			return printJSON(card)
		}
		return printCardsTable([]Card{card})
	default:
		return fmt.Errorf("unknown cards subcommand %q", args[0])
	}
}

func runComments(client *Client, cfg Config, args []string) error {
	if len(args) == 0 {
		printCommentsHelp()
		return nil
	}

	switch args[0] {
	case "-h", "--help", "help":
		printCommentsHelp()
		return nil
	case "list":
		fs := flag.NewFlagSet("comments list", flag.ContinueOnError)
		fs.SetOutput(io.Discard)
		var cardID string
		limit := 100
		fs.StringVar(&cardID, "card", "", "Card id")
		fs.IntVar(&limit, "limit", limit, "Max comments to return")
		if err := parseFlagSet(fs, args[1:], printCommentsHelp); err != nil {
			return err
		}
		if strings.TrimSpace(cardID) == "" {
			return errors.New("comments list requires --card")
		}

		query := url.Values{}
		query.Set("filter", "commentCard")
		query.Set("fields", "data,date,type")
		query.Set("memberCreator_fields", "username,fullName")
		query.Set("limit", fmt.Sprintf("%d", limit))

		var actions []CommentAction
		if err := client.do(http.MethodGet, "/1/cards/"+url.PathEscape(cardID)+"/actions", query, nil, &actions); err != nil {
			return err
		}
		if cfg.JSON {
			return printJSON(actions)
		}
		return printCommentsTable(actions)

	case "add":
		fs := flag.NewFlagSet("comments add", flag.ContinueOnError)
		fs.SetOutput(io.Discard)
		var cardID, text string
		fs.StringVar(&cardID, "card", "", "Card id")
		fs.StringVar(&text, "text", "", "Comment text")
		if err := parseFlagSet(fs, args[1:], printCommentsHelp); err != nil {
			return err
		}
		if strings.TrimSpace(cardID) == "" || strings.TrimSpace(text) == "" {
			return errors.New("comments add requires --card and --text")
		}

		form := url.Values{}
		form.Set("text", text)
		var created CommentAction
		if err := client.do(http.MethodPost, "/1/cards/"+url.PathEscape(cardID)+"/actions/comments", nil, form, &created); err != nil {
			return err
		}
		if cfg.JSON {
			return printJSON(created)
		}
		return printCommentsTable([]CommentAction{created})
	default:
		return fmt.Errorf("unknown comments subcommand %q", args[0])
	}
}

func runChecklists(client *Client, cfg Config, args []string) error {
	if len(args) == 0 {
		printChecklistsHelp()
		return nil
	}

	switch args[0] {
	case "-h", "--help", "help":
		printChecklistsHelp()
		return nil
	case "list":
		fs := flag.NewFlagSet("checklists list", flag.ContinueOnError)
		fs.SetOutput(io.Discard)
		var cardID string
		fs.StringVar(&cardID, "card", "", "Card id")
		if err := parseFlagSet(fs, args[1:], printChecklistsHelp); err != nil {
			return err
		}
		if strings.TrimSpace(cardID) == "" {
			return errors.New("checklists list requires --card")
		}

		query := url.Values{}
		query.Set("checkItems", "all")
		query.Set("checkItem_fields", "name,state,pos")
		var checklists []Checklist
		if err := client.do(http.MethodGet, "/1/cards/"+url.PathEscape(cardID)+"/checklists", query, nil, &checklists); err != nil {
			return err
		}
		if cfg.JSON {
			return printJSON(checklists)
		}
		return printChecklistsTable(checklists)

	case "create":
		fs := flag.NewFlagSet("checklists create", flag.ContinueOnError)
		fs.SetOutput(io.Discard)
		var cardID, name string
		fs.StringVar(&cardID, "card", "", "Card id")
		fs.StringVar(&name, "name", "", "Checklist name")
		if err := parseFlagSet(fs, args[1:], printChecklistsHelp); err != nil {
			return err
		}
		if strings.TrimSpace(cardID) == "" || strings.TrimSpace(name) == "" {
			return errors.New("checklists create requires --card and --name")
		}

		form := url.Values{}
		form.Set("name", name)
		var checklist Checklist
		if err := client.do(http.MethodPost, "/1/cards/"+url.PathEscape(cardID)+"/checklists", nil, form, &checklist); err != nil {
			return err
		}
		if cfg.JSON {
			return printJSON(checklist)
		}
		return printChecklistsTable([]Checklist{checklist})

	case "add-item":
		fs := flag.NewFlagSet("checklists add-item", flag.ContinueOnError)
		fs.SetOutput(io.Discard)
		var checklistID, name string
		var checked bool
		fs.StringVar(&checklistID, "checklist", "", "Checklist id")
		fs.StringVar(&name, "name", "", "Item name")
		fs.BoolVar(&checked, "checked", false, "Create item as checked")
		if err := parseFlagSet(fs, args[1:], printChecklistsHelp); err != nil {
			return err
		}
		if strings.TrimSpace(checklistID) == "" || strings.TrimSpace(name) == "" {
			return errors.New("checklists add-item requires --checklist and --name")
		}

		form := url.Values{}
		form.Set("name", name)
		if checked {
			form.Set("checked", "true")
		}
		var item ChecklistItem
		if err := client.do(http.MethodPost, "/1/checklists/"+url.PathEscape(checklistID)+"/checkItems", nil, form, &item); err != nil {
			return err
		}
		if cfg.JSON {
			return printJSON(item)
		}
		return printChecklistItemsTable([]ChecklistItem{item})

	case "set-item":
		fs := flag.NewFlagSet("checklists set-item", flag.ContinueOnError)
		fs.SetOutput(io.Discard)
		var cardID, itemID, state string
		fs.StringVar(&cardID, "card", "", "Card id")
		fs.StringVar(&itemID, "item", "", "Checklist item id")
		fs.StringVar(&state, "state", "", "State: complete|incomplete")
		if err := parseFlagSet(fs, args[1:], printChecklistsHelp); err != nil {
			return err
		}
		state = strings.TrimSpace(strings.ToLower(state))
		if strings.TrimSpace(cardID) == "" || strings.TrimSpace(itemID) == "" || state == "" {
			return errors.New("checklists set-item requires --card, --item, and --state")
		}
		if state != "complete" && state != "incomplete" {
			return errors.New("--state must be complete or incomplete")
		}

		form := url.Values{}
		form.Set("state", state)
		var updated ChecklistItem
		if err := client.do(http.MethodPut, "/1/cards/"+url.PathEscape(cardID)+"/checkItem/"+url.PathEscape(itemID), nil, form, &updated); err != nil {
			return err
		}
		if cfg.JSON {
			return printJSON(updated)
		}
		return printChecklistItemsTable([]ChecklistItem{updated})
	default:
		return fmt.Errorf("unknown checklists subcommand %q", args[0])
	}
}

func fetchBoardLists(client *Client, boardID string) ([]TrelloList, error) {
	query := url.Values{}
	query.Set("fields", "id,name,closed,pos")
	var lists []TrelloList
	if err := client.do(http.MethodGet, "/1/boards/"+url.PathEscape(boardID)+"/lists", query, nil, &lists); err != nil {
		return nil, err
	}
	return lists, nil
}

func resolveListID(client *Client, boardID, listID, listName string) (string, error) {
	listID = strings.TrimSpace(listID)
	listName = strings.TrimSpace(listName)
	boardID = strings.TrimSpace(boardID)
	if listID != "" {
		return listID, nil
	}
	if listName == "" {
		return "", errors.New("missing list target: provide --list or --list-name")
	}
	if boardID == "" {
		return "", errors.New("--board is required with --list-name")
	}

	lists, err := fetchBoardLists(client, boardID)
	if err != nil {
		return "", err
	}

	target := strings.ToLower(listName)
	exactMatches := make([]TrelloList, 0)
	partialMatches := make([]TrelloList, 0)
	for _, l := range lists {
		name := strings.ToLower(l.Name)
		if name == target {
			exactMatches = append(exactMatches, l)
			continue
		}
		if strings.Contains(name, target) {
			partialMatches = append(partialMatches, l)
		}
	}
	if len(exactMatches) == 1 {
		return exactMatches[0].ID, nil
	}
	if len(exactMatches) > 1 {
		return "", fmt.Errorf("list name %q is ambiguous on board %q (%d exact matches)", listName, boardID, len(exactMatches))
	}
	if len(partialMatches) == 1 {
		return partialMatches[0].ID, nil
	}
	if len(partialMatches) > 1 {
		return "", fmt.Errorf("list name %q is ambiguous on board %q (%d partial matches)", listName, boardID, len(partialMatches))
	}
	return "", fmt.Errorf("list name %q not found on board %q", listName, boardID)
}

func shouldSkipAuthForHelp(args []string) bool {
	if len(args) == 0 {
		return true
	}
	first := strings.TrimSpace(strings.ToLower(args[0]))
	if first == "help" || first == "-h" || first == "--help" {
		return true
	}
	for _, a := range args {
		if a == "-h" || a == "--help" {
			return true
		}
	}
	return false
}

func parseFlagSet(fs *flag.FlagSet, args []string, helpFn func()) error {
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			helpFn()
			return errHelpDisplayed
		}
		return err
	}
	return nil
}

func printJSON(v any) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

func printBoardsTable(boards []Board) error {
	if len(boards) == 0 {
		fmt.Println("No boards found.")
		return nil
	}
	tw := tabwriter.NewWriter(os.Stdout, 2, 8, 2, ' ', 0)
	fmt.Fprintln(tw, "ID\tNAME\tCLOSED\tURL")
	for _, b := range boards {
		fmt.Fprintf(tw, "%s\t%s\t%t\t%s\n", b.ID, b.Name, b.Closed, b.URL)
	}
	return tw.Flush()
}

func printListsTable(lists []TrelloList) error {
	if len(lists) == 0 {
		fmt.Println("No lists found.")
		return nil
	}
	tw := tabwriter.NewWriter(os.Stdout, 2, 8, 2, ' ', 0)
	fmt.Fprintln(tw, "ID\tNAME\tCLOSED")
	for _, l := range lists {
		fmt.Fprintf(tw, "%s\t%s\t%t\n", l.ID, l.Name, l.Closed)
	}
	return tw.Flush()
}

func printCardsTable(cards []Card) error {
	if len(cards) == 0 {
		fmt.Println("No cards found.")
		return nil
	}
	tw := tabwriter.NewWriter(os.Stdout, 2, 8, 2, ' ', 0)
	fmt.Fprintln(tw, "ID\tNAME\tLIST\tDUE\tCLOSED\tURL")
	for _, c := range cards {
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%t\t%s\n", c.ID, c.Name, c.IDList, c.Due, c.Closed, firstNonEmpty(c.ShortURL, c.URL))
	}
	return tw.Flush()
}

func printCommentsTable(actions []CommentAction) error {
	if len(actions) == 0 {
		fmt.Println("No comments found.")
		return nil
	}
	tw := tabwriter.NewWriter(os.Stdout, 2, 8, 2, ' ', 0)
	fmt.Fprintln(tw, "ID\tDATE\tAUTHOR\tCOMMENT")
	for _, a := range actions {
		author := strings.TrimSpace(firstNonEmpty(a.MemberCreator.FullName, a.MemberCreator.Username))
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n", a.ID, a.Date, author, a.Data.Text)
	}
	return tw.Flush()
}

func printChecklistsTable(checklists []Checklist) error {
	if len(checklists) == 0 {
		fmt.Println("No checklists found.")
		return nil
	}
	tw := tabwriter.NewWriter(os.Stdout, 2, 8, 2, ' ', 0)
	fmt.Fprintln(tw, "CHECKLIST_ID\tCHECKLIST_NAME\tITEM_ID\tITEM_STATE\tITEM_NAME")
	for _, cl := range checklists {
		if len(cl.CheckItems) == 0 {
			fmt.Fprintf(tw, "%s\t%s\t\t\t\n", cl.ID, cl.Name)
			continue
		}
		for _, item := range cl.CheckItems {
			fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\n", cl.ID, cl.Name, item.ID, item.State, item.Name)
		}
	}
	return tw.Flush()
}

func printChecklistItemsTable(items []ChecklistItem) error {
	if len(items) == 0 {
		fmt.Println("No checklist items found.")
		return nil
	}
	tw := tabwriter.NewWriter(os.Stdout, 2, 8, 2, ' ', 0)
	fmt.Fprintln(tw, "ITEM_ID\tSTATE\tNAME")
	for _, item := range items {
		fmt.Fprintf(tw, "%s\t%s\t%s\n", item.ID, item.State, item.Name)
	}
	return tw.Flush()
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

func printRootHelp() {
	fmt.Print(`trelli - Efficient Trello CLI

Usage:
  trelli [global options] <command> <subcommand> [options]
  trelli help [command]
  trelli version

Global options:
  --key <key>       Trello API key (default: TRELLO_API_KEY)
  --token <token>   Trello token (default: TRELLO_TOKEN)
  --board <id>      Default board id/shortLink (default: TRELLO_BOARD_ID or XobnRsYv)
  --json            Output raw JSON
  -h, --help        Show help

Commands:
  boards      Board-level commands
  lists       List-level commands
  cards       Card-level commands
  comments    Card comment commands
  checklists  Card checklist commands
  help        Show help for command
  version     Show CLI version

Subcommands:
  boards list
  lists list
  cards list | show | create | move | archive
  comments list | add
  checklists list | create | add-item | set-item

Detailed usage:
  trelli boards list [--filter <name-substring>]
  trelli lists list [--board <boardIdOrShortLink>]
  trelli cards list --list <listId> [--limit <n>]
  trelli cards list --list-name <name> [--board <boardIdOrShortLink>] [--limit <n>]
  trelli cards show --card <cardId>
  trelli cards create (--list <listId> | --list-name <name>) --name <title> [--desc <text>] [--due <iso8601>] [--labels <id1,id2>] [--members <id1,id2>] [--board <boardIdOrShortLink>]
  trelli cards move --card <cardId> (--list <listId> | --list-name <name>) [--board <boardIdOrShortLink>]
  trelli cards archive --card <cardId>
  trelli comments list --card <cardId> [--limit <n>]
  trelli comments add --card <cardId> --text <comment>
  trelli checklists list --card <cardId>
  trelli checklists create --card <cardId> --name <checklistName>
  trelli checklists add-item --checklist <checklistId> --name <itemName> [--checked]
  trelli checklists set-item --card <cardId> --item <itemId> --state <complete|incomplete>

Examples:
  trelli boards list
  trelli lists list --board XobnRsYv
  trelli cards create --list-name "To Do" --name "Build CLI" --desc "Initial implementation"
  trelli comments add --card <cardId> --text "Started implementation"
  trelli checklists add-item --checklist <checklistId> --name "Write tests"

For command help:
  trelli help cards
  trelli cards --help
`)
}

func printBoardsHelp() {
	fmt.Print(`Usage:
  trelli boards list [--filter <name-substring>]

Description:
  List boards visible to the authenticated user.

Options:
  --filter <text>   Case-insensitive board name filter
  --json            Output raw JSON
`)
}

func printListsHelp() {
	fmt.Print(`Usage:
  trelli lists list [--board <boardIdOrShortLink>]

Description:
  List all lists for a board. Defaults to --board from global flag or TRELLO_BOARD_ID.

Options:
  --board <id>      Board id or shortLink
  --json            Output raw JSON
`)
}

func printCardsHelp() {
	fmt.Print(`Usage:
  trelli cards list --list <listId> [--limit <n>]
  trelli cards list --list-name <name> [--board <boardIdOrShortLink>] [--limit <n>]
  trelli cards show --card <cardId>
  trelli cards create (--list <listId> | --list-name <name>) --name <title> [--desc <text>] [--due <iso8601>] [--labels <id1,id2>] [--members <id1,id2>] [--board <boardIdOrShortLink>]
  trelli cards move --card <cardId> (--list <listId> | --list-name <name>) [--board <boardIdOrShortLink>]
  trelli cards archive --card <cardId>

Description:
  Manage cards: list, create, inspect, move, and archive.

Options:
  --list <id>       List id
  --list-name <n>   List name (resolved on board)
  --board <id>      Board id or shortLink (used with --list-name)
  --card <id>       Card id
  --name <text>     Card title (create)
  --desc <text>     Card description (create)
  --due <iso8601>   Card due date/time, e.g. 2026-02-14T18:00:00Z
  --labels <ids>    Comma-separated label ids
  --members <ids>   Comma-separated member ids
  --limit <n>       Number of cards for list operation (default 100)
  --json            Output raw JSON
`)
}

func printCommentsHelp() {
	fmt.Print(`Usage:
  trelli comments list --card <cardId> [--limit <n>]
  trelli comments add --card <cardId> --text <comment>

Description:
  Read or add comments on a card.

Options:
  --card <id>       Card id
  --text <text>     Comment body
  --limit <n>       Number of comments to fetch (default 100)
  --json            Output raw JSON
`)
}

func printChecklistsHelp() {
	fmt.Print(`Usage:
  trelli checklists list --card <cardId>
  trelli checklists create --card <cardId> --name <checklistName>
  trelli checklists add-item --checklist <checklistId> --name <itemName> [--checked]
  trelli checklists set-item --card <cardId> --item <itemId> --state <complete|incomplete>

Description:
  Manage card checklists and items.

Options:
  --card <id>          Card id
  --checklist <id>     Checklist id
  --item <id>          Checklist item id
  --name <text>        Checklist or item name
  --checked            Create item as checked
  --state <state>      complete|incomplete
  --json               Output raw JSON
`)
}

func printCommandHelp(cmd string) {
	switch cmd {
	case "boards":
		printBoardsHelp()
	case "lists":
		printListsHelp()
	case "cards":
		printCardsHelp()
	case "comments":
		printCommentsHelp()
	case "checklists":
		printChecklistsHelp()
	default:
		printRootHelp()
	}
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format, args...)
	os.Exit(1)
}
