package main

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"log"
	"math/rand"

	"github.com/zappabad/stockcraft/internal/engine"

	tea "github.com/charmbracelet/bubbletea"
)

/*
Adapt these to your code:

- If you already have `type Market struct { ... }` with methods:
    GetTickers() []Ticker
    GetPrice(t Ticker) (price int, ok bool, err error)
  then your *Market will satisfy MarketAPI automatically.

- If your signatures differ, tweak MarketAPI / fetchSnapshot accordingly.
*/

type MarketAPI interface {
	GetTickers() []engine.Ticker
	GetPrice(t engine.Ticker) (price engine.PriceTicks, ok bool, err error)
}

type row struct {
	name  string
	price string
}

type snapshotMsg struct {
	rows []row
	ts   time.Time
	err  error
}

type tickMsg time.Time

type model struct {
	market      MarketAPI
	interval    time.Duration
	rows        []row
	lastUpdated time.Time
	lastErr     error
}

func NewModel(m MarketAPI, interval time.Duration) model {
	return model{
		market:   m,
		interval: interval,
	}
}

func (m model) Init() tea.Cmd {
	// Fetch immediately, and start the periodic tick.
	return tea.Batch(fetchSnapshot(m.market), tick(m.interval))
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c", "esc":
			return m, tea.Quit
		}

	case tickMsg:
		// Schedule next tick and fetch a fresh snapshot.
		return m, tea.Batch(tick(m.interval), fetchSnapshot(m.market))

	case snapshotMsg:
		m.rows = msg.rows
		m.lastUpdated = msg.ts
		m.lastErr = msg.err
		return m, nil
	}

	return m, nil
}

func (m model) View() string {
	var b strings.Builder

	b.WriteString("Market Snapshot (press q to quit)\n\n")

	if !m.lastUpdated.IsZero() {
		b.WriteString(fmt.Sprintf("Last update: %s\n\n", m.lastUpdated.Format(time.RFC3339)))
	}

	if m.lastErr != nil {
		b.WriteString(fmt.Sprintf("Warning: last update had errors: %v\n\n", m.lastErr))
	}

	// Simple fixed-width table
	b.WriteString(fmt.Sprintf("%-12s %s\n", "TICKER", "PRICE"))
	b.WriteString(strings.Repeat("-", 28))
	b.WriteString("\n")

	for _, r := range m.rows {
		b.WriteString(fmt.Sprintf("%-12s %s\n", r.name, r.price))
	}

	return b.String()
}

func tick(d time.Duration) tea.Cmd {
	return tea.Tick(d, func(t time.Time) tea.Msg { return tickMsg(t) })
}

func fetchSnapshot(market MarketAPI) tea.Cmd {
	return func() tea.Msg {
		tickers := market.GetTickers()

		// Stable ordering so the UI doesn’t jump around.
		sort.Slice(tickers, func(i, j int) bool { return tickers[i].Name < tickers[j].Name })

		rows := make([]row, 0, len(tickers))
		var firstErr error

		for _, t := range tickers {
			price, ok, err := market.GetPrice(t)
			if err != nil && firstErr == nil {
				firstErr = err
			}

			switch {
			case err != nil:
				rows = append(rows, row{name: t.Name, price: "ERR"})
			case !ok:
				rows = append(rows, row{name: t.Name, price: "N/A"})
			default:
				rows = append(rows, row{name: t.Name, price: fmt.Sprintf("%d", price)})
			}
		}

		return snapshotMsg{rows: rows, ts: time.Now(), err: firstErr}
	}
}

func RunMarketUI(m MarketAPI, interval time.Duration) error {
	p := tea.NewProgram(NewModel(m, interval))
	_, err := p.Run()
	return err
}

func main() {
	tickers := []engine.Ticker{
		{ID: 1, Name: "AAPL", Decimals: 2},
		{ID: 2, Name: "GOOGL", Decimals: 2},
		{ID: 3, Name: "NVDA", Decimals: 2},
		{ID: 4, Name: "AMZN", Decimals: 2},
		{ID: 5, Name: "MSFT", Decimals: 2},
		{ID: 6, Name: "TSLA", Decimals: 2},
		{ID: 7, Name: "META", Decimals: 2},
		{ID: 8, Name: "NFLX", Decimals: 2},
		{ID: 9, Name: "BABA", Decimals: 2},
		{ID: 10, Name: "INTC", Decimals: 2},
	}

	// 1. Create a basic market.
	market := engine.NewMarket(tickers)

	// 2. Create some traders.
	// TODO: Replace these with real strategy types (frequent, swing, news-based).
	var total_traders int64 = 500
	traders := []engine.Trader{}

	for i := range total_traders {
		traderSeed := rand.New(rand.NewSource(int64(i + 69420)))
		traders = append(traders, engine.NewRandomTrader(i, []string{"BAR"}, traderSeed))
	}

	// Simple sanity check: ensure we have at least one trader.
	if len(traders) == 0 {
		log.Fatal("no traders configured")
	}

	// 3. Generate News Engine
	newsEngine := engine.NewNewsEngine()

	// 4. Wire everything into a simulation.
	sim := engine.NewSimulation(market, traders, newsEngine)

	_ = RunMarketUI(&sim.Market, 500*time.Millisecond)
}
