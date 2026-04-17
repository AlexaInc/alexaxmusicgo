package dashboard

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"runtime"
	"time"

	"alexamusic/internal/config"
	"alexamusic/internal/db"
	"alexamusic/internal/queue"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/process"
)

var bootTime = time.Now()

// Stats holds the live bot statistics.
type Stats struct {
	BotName      string  `json:"bot_name"`
	Uptime       string  `json:"uptime"`
	ActiveCalls  int     `json:"active_calls"`
	TotalChats   int     `json:"total_chats"`
	TotalUsers   int     `json:"total_users"`
	MemUsedMB    float64 `json:"mem_used_mb"`
	MemTotalGB   float64 `json:"mem_total_gb"`
	CPUPercent   float64 `json:"cpu_percent"`
	DiskUsedGB   float64 `json:"disk_used_gb"`
	DiskTotalGB  float64 `json:"disk_total_gb"`
	GoVersion    string  `json:"go_version"`
	BotVersion   string  `json:"bot_version"`
}

func collectStats(cfg *config.Config) Stats {
	uptime := time.Since(bootTime)
	hours := int(uptime.Hours())
	mins := int(uptime.Minutes()) % 60
	secs := int(uptime.Seconds()) % 60

	// Memory
	vm, _ := mem.VirtualMemory()
	memUsedMB := float64(vm.Used) / (1024 * 1024)
	memTotalGB := float64(vm.Total) / (1024 * 1024 * 1024)

	// CPU
	cpuPct, _ := cpu.Percent(0, false)
	cpuUsage := 0.0
	if len(cpuPct) > 0 {
		cpuUsage = cpuPct[0]
	}

	// Disk
	du, _ := disk.Usage("/")
	diskUsedGB, diskTotalGB := 0.0, 0.0
	if du != nil {
		diskUsedGB = float64(du.Used) / (1024 * 1024 * 1024)
		diskTotalGB = float64(du.Total) / (1024 * 1024 * 1024)
	}

	// Active calls
	activeCalls := 0
	if db.DB != nil {
		activeCalls = db.DB.GetActiveCallsCount()
	}
	_ = queue.Q // ensure package is linked

	return Stats{
		BotName:     cfg.MusicBotName,
		Uptime:      fmt.Sprintf("%02d:%02d:%02d", hours, mins, secs),
		ActiveCalls: activeCalls,
		TotalChats:  func() int { if db.DB != nil { return len(db.DB.GetChats()) }; return 0 }(),
		TotalUsers:  func() int { if db.DB != nil { return len(db.DB.GetUsers()) }; return 0 }(),
		MemUsedMB:   memUsedMB,
		MemTotalGB:  memTotalGB,
		CPUPercent:  cpuUsage,
		DiskUsedGB:  diskUsedGB,
		DiskTotalGB: diskTotalGB,
		GoVersion:   runtime.Version(),
		BotVersion:  cfg.Version,
	}
}

// Start launches the HTTP dashboard server on the configured port.
func Start(cfg *config.Config) {
	mux := http.NewServeMux()

	// Serve the dashboard HTML
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte(dashboardHTML(cfg)))
	})

	// Stats API endpoint
	mux.HandleFunc("/api/stats", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		stats := collectStats(cfg)
		json.NewEncoder(w).Encode(stats)
	})

	// Health check
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Alive"))
	})

	addr := fmt.Sprintf("0.0.0.0:%d", cfg.Port)
	log.Printf("[dashboard] Listening on %s", addr)
	go func() {
		if err := http.ListenAndServe(addr, mux); err != nil {
			log.Printf("[dashboard] Server error: %v", err)
		}
	}()
}

// processMemMB returns current process RSS in MB.
func processMemMB() float64 {
	p, err := process.NewProcess(int32(0))
	if err != nil {
		return 0
	}
	info, err := p.MemoryInfo()
	if err != nil {
		return 0
	}
	return float64(info.RSS) / (1024 * 1024)
}

func dashboardHTML(cfg *config.Config) string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8"/>
<meta name="viewport" content="width=device-width,initial-scale=1"/>
<meta name="description" content="%s – Telegram Music Bot Dashboard"/>
<title>%s Dashboard</title>
<link rel="preconnect" href="https://fonts.googleapis.com"/>
<link href="https://fonts.googleapis.com/css2?family=Inter:wght@300;400;600;700;800&display=swap" rel="stylesheet"/>
<style>
  *,*::before,*::after{box-sizing:border-box;margin:0;padding:0}
  :root{
    --bg:#0a0a0f;--surface:#13131c;--card:#1a1a28;--accent:#7c3aed;
    --accent2:#06b6d4;--text:#e2e8f0;--muted:#64748b;--border:#252535;
    --green:#10b981;--yellow:#f59e0b;--red:#ef4444;
  }
  body{background:var(--bg);color:var(--text);font-family:'Inter',sans-serif;
       min-height:100vh;padding:2rem;background-image:radial-gradient(ellipse at 20%% 50%%,rgba(124,58,237,.08) 0%%,transparent 60%%),
       radial-gradient(ellipse at 80%% 20%%,rgba(6,182,212,.06) 0%%,transparent 50%%)}
  .header{text-align:center;margin-bottom:3rem}
  .header h1{font-size:2.4rem;font-weight:800;background:linear-gradient(135deg,var(--accent),var(--accent2));
             -webkit-background-clip:text;-webkit-text-fill-color:transparent;letter-spacing:-1px}
  .header p{color:var(--muted);margin-top:.4rem;font-size:.95rem}
  .badge{display:inline-flex;align-items:center;gap:.4rem;background:rgba(16,185,129,.15);
         color:var(--green);border:1px solid rgba(16,185,129,.3);border-radius:999px;
         padding:.25rem .8rem;font-size:.8rem;font-weight:600;margin-top:.8rem}
  .badge::before{content:'';width:8px;height:8px;border-radius:50%%;background:var(--green);
                 animation:pulse 2s infinite}
  @keyframes pulse{0%%,100%%{opacity:1;transform:scale(1)}50%%{opacity:.6;transform:scale(.8)}}
  .grid{display:grid;grid-template-columns:repeat(auto-fit,minmax(200px,1fr));gap:1.2rem;margin-bottom:2rem}
  .card{background:var(--card);border:1px solid var(--border);border-radius:16px;padding:1.4rem;
        transition:transform .2s,box-shadow .2s;position:relative;overflow:hidden}
  .card:hover{transform:translateY(-3px);box-shadow:0 12px 40px rgba(0,0,0,.4)}
  .card::after{content:'';position:absolute;inset:0;border-radius:16px;background:linear-gradient(135deg,rgba(255,255,255,.02),transparent)}
  .card-icon{width:40px;height:40px;border-radius:10px;display:flex;align-items:center;justify-content:center;
             font-size:1.2rem;margin-bottom:.8rem}
  .card-val{font-size:2rem;font-weight:800;line-height:1}
  .card-label{color:var(--muted);font-size:.8rem;margin-top:.3rem;font-weight:500;text-transform:uppercase;letter-spacing:.05em}
  .section{background:var(--card);border:1px solid var(--border);border-radius:16px;padding:1.6rem;margin-bottom:1.5rem}
  .section h2{font-size:1rem;font-weight:700;color:var(--muted);text-transform:uppercase;letter-spacing:.1em;margin-bottom:1.2rem}
  .bar-wrap{display:flex;flex-direction:column;gap:.8rem}
  .bar-row{display:flex;flex-direction:column;gap:.3rem}
  .bar-meta{display:flex;justify-content:space-between;font-size:.82rem}
  .bar-bg{height:8px;background:var(--border);border-radius:4px;overflow:hidden}
  .bar-fill{height:100%%;border-radius:4px;transition:width 1s ease}
  .info-grid{display:grid;grid-template-columns:1fr 1fr;gap:.6rem}
  .info-row{display:flex;justify-content:space-between;padding:.5rem .8rem;background:var(--surface);
            border-radius:8px;font-size:.85rem}
  .info-row span:first-child{color:var(--muted)}
  .info-row span:last-child{font-weight:600}
  .update-time{text-align:center;color:var(--muted);font-size:.78rem;margin-top:1rem}
  @media(max-width:600px){body{padding:1rem}.header h1{font-size:1.8rem}}
</style>
</head>
<body>
<div class="header">
  <h1>🎵 %s</h1>
  <p>Telegram Music Bot – Live Dashboard</p>
  <span class="badge">LIVE</span>
</div>

<div class="grid">
  <div class="card">
    <div class="card-icon" style="background:rgba(124,58,237,.15)">📞</div>
    <div class="card-val" id="active_calls" style="color:#a78bfa">—</div>
    <div class="card-label">Active Calls</div>
  </div>
  <div class="card">
    <div class="card-icon" style="background:rgba(6,182,212,.15)">💬</div>
    <div class="card-val" id="total_chats" style="color:#67e8f9">—</div>
    <div class="card-label">Total Chats</div>
  </div>
  <div class="card">
    <div class="card-icon" style="background:rgba(16,185,129,.15)">👥</div>
    <div class="card-val" id="total_users" style="color:#6ee7b7">—</div>
    <div class="card-label">Total Users</div>
  </div>
  <div class="card">
    <div class="card-icon" style="background:rgba(245,158,11,.15)">⏱️</div>
    <div class="card-val" id="uptime" style="color:#fcd34d;font-size:1.4rem">—</div>
    <div class="card-label">Uptime</div>
  </div>
</div>

<div class="section">
  <h2>Resource Usage</h2>
  <div class="bar-wrap">
    <div class="bar-row">
      <div class="bar-meta"><span>CPU</span><span id="cpu_pct">—</span></div>
      <div class="bar-bg"><div class="bar-fill" id="cpu_bar" style="background:linear-gradient(90deg,#7c3aed,#a78bfa);width:0%%"></div></div>
    </div>
    <div class="bar-row">
      <div class="bar-meta"><span>Memory</span><span id="mem_info">—</span></div>
      <div class="bar-bg"><div class="bar-fill" id="mem_bar" style="background:linear-gradient(90deg,#0891b2,#67e8f9);width:0%%"></div></div>
    </div>
    <div class="bar-row">
      <div class="bar-meta"><span>Disk</span><span id="disk_info">—</span></div>
      <div class="bar-bg"><div class="bar-fill" id="disk_bar" style="background:linear-gradient(90deg,#059669,#6ee7b7);width:0%%"></div></div>
    </div>
  </div>
</div>

<div class="section">
  <h2>Bot Info</h2>
  <div class="info-grid">
    <div class="info-row"><span>Bot Name</span><span id="bot_name">—</span></div>
    <div class="info-row"><span>Version</span><span id="bot_version">—</span></div>
    <div class="info-row"><span>Go Version</span><span id="go_version">—</span></div>
    <div class="info-row"><span>Status</span><span style="color:var(--green)">Online ✓</span></div>
  </div>
</div>

<p class="update-time">Last updated: <span id="last_updated">—</span></p>

<script>
async function fetchStats(){
  try{
    const r=await fetch('/api/stats');
    const d=await r.json();
    document.getElementById('active_calls').textContent=d.active_calls;
    document.getElementById('total_chats').textContent=d.total_chats.toLocaleString();
    document.getElementById('total_users').textContent=d.total_users.toLocaleString();
    document.getElementById('uptime').textContent=d.uptime;
    document.getElementById('cpu_pct').textContent=d.cpu_percent.toFixed(1)+'%%';
    document.getElementById('cpu_bar').style.width=Math.min(d.cpu_percent,100)+'%%';
    const memPct=d.mem_total_gb>0?(d.mem_used_mb/(d.mem_total_gb*1024)*100):0;
    document.getElementById('mem_info').textContent=d.mem_used_mb.toFixed(0)+'MB / '+d.mem_total_gb.toFixed(1)+'GB';
    document.getElementById('mem_bar').style.width=Math.min(memPct,100)+'%%';
    const diskPct=d.disk_total_gb>0?(d.disk_used_gb/d.disk_total_gb*100):0;
    document.getElementById('disk_info').textContent=d.disk_used_gb.toFixed(1)+'GB / '+d.disk_total_gb.toFixed(1)+'GB';
    document.getElementById('disk_bar').style.width=Math.min(diskPct,100)+'%%';
    document.getElementById('bot_name').textContent=d.bot_name;
    document.getElementById('bot_version').textContent=d.bot_version;
    document.getElementById('go_version').textContent=d.go_version;
    document.getElementById('last_updated').textContent=new Date().toLocaleTimeString();
  }catch(e){console.error(e)}
}
fetchStats();
setInterval(fetchStats,5000);
</script>
</body>
</html>
`, cfg.MusicBotName, cfg.MusicBotName, cfg.MusicBotName)
}
