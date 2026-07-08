import React, { useState, useEffect, useCallback, useRef } from 'react';
import { 
  Activity, 
  Layers, 
  PlusCircle, 
  Terminal, 
  List, 
  AlertTriangle, 
  Settings, 
  Play, 
  Power, 
  Trash2, 
  Search, 
  Server, 
  Database, 
  Cpu, 
  RefreshCw,
  Clock
} from 'lucide-react';

// API Configurations
const API_BASE = `http://${window.location.hostname}:8080`;

interface Sandbox {
  id: number;
  sandbox_name: string;
  owner: string;
  status: string;
  created_at: string;
  updated_at: string;
}

interface SandboxLog {
  id: number;
  sandbox_id: number;
  message: string;
  log_level: string;
  created_at: string;
}

export default function App() {
  const [activeTab, setActiveTab] = useState<'dashboard' | 'sandboxes' | 'create' | 'logs' | 'failures'>('dashboard');
  const [sandboxes, setSandboxes] = useState<Sandbox[]>([]);
  const [dbLogs, setDbLogs] = useState<SandboxLog[]>([]);
  const [selectedSandbox, setSelectedSandbox] = useState<Sandbox | null>(null);
  
  // Terminal commands state
  const [commandInput, setCommandInput] = useState('');
  const [terminalOutputs, setTerminalOutputs] = useState<string[]>([]);
  const terminalEndRef = useRef<HTMLDivElement>(null);

  // Health and polling metrics
  const [backendHealth, setBackendHealth] = useState<'UP' | 'DOWN' | 'PENDING'>('PENDING');
  const [apiLatencyHistory, setApiLatencyHistory] = useState<number[]>(new Array(15).fill(0));
  const [pollingActive] = useState(true);

  // Failure states
  const [simApiDelay, setSimApiDelay] = useState(0);
  const [simDbDelay, setSimDbDelay] = useState(0);
  const [simDbFailure, setSimDbFailure] = useState(false);
  const [simVsiTimeout, setSimVsiTimeout] = useState(false);
  const [simRandomErrors, setSimRandomErrors] = useState(false);
  const [simHighCpu, setSimHighCpu] = useState(false);
  const [simHighMemory, setSimHighMemory] = useState(0);

  // Create form state
  const [newName, setNewName] = useState('');
  const [newOwner, setNewOwner] = useState('');
  const [isSubmitting, setIsSubmitting] = useState(false);

  // Log filter state
  const [logFilterId, setLogFilterId] = useState('');
  const [logFilterLevel, setLogFilterLevel] = useState('');

  // Auto-scroll terminal
  useEffect(() => {
    terminalEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, [terminalOutputs]);

  // Load sandboxes
  const fetchSandboxes = useCallback(async () => {
    try {
      const start = performance.now();
      const res = await fetch(`${API_BASE}/sandbox`);
      const latency = performance.now() - start;
      updateLatencyHistory(latency);
      if (res.ok) {
        const data = await res.json();
        setSandboxes(data || []);
        setBackendHealth('UP');
      } else {
        setBackendHealth('DOWN');
      }
    } catch {
      setBackendHealth('DOWN');
    }
  }, []);

  // Fetch db logs
  const fetchLogs = useCallback(async () => {
    try {
      let url = `${API_BASE}/logs`;
      const queryParams: string[] = [];
      if (logFilterId) queryParams.push(`sandbox_id=${logFilterId}`);
      if (logFilterLevel) queryParams.push(`level=${logFilterLevel}`);
      if (queryParams.length > 0) {
        url += '?' + queryParams.join('&');
      }
      
      const res = await fetch(url);
      if (res.ok) {
        const data = await res.json();
        setDbLogs(data || []);
      }
    } catch (e) {
      console.error(e);
    }
  }, [logFilterId, logFilterLevel]);

  // Latency graph updater
  const updateLatencyHistory = (newVal: number) => {
    setApiLatencyHistory(prev => {
      const slice = prev.slice(1);
      return [...slice, Math.round(newVal)];
    });
  };

  // Ping backend for live latency metrics
  useEffect(() => {
    if (!pollingActive) return;
    const interval = setInterval(async () => {
      try {
        const start = performance.now();
        const res = await fetch(`${API_BASE}/health`);
        const latency = performance.now() - start;
        updateLatencyHistory(latency);
        if (res.ok) {
          setBackendHealth('UP');
        } else {
          setBackendHealth('DOWN');
        }
      } catch {
        setBackendHealth('DOWN');
        updateLatencyHistory(0);
      }
    }, 1500);

    return () => clearInterval(interval);
  }, [pollingActive]);

  // Load initial content
  useEffect(() => {
    fetchSandboxes();
    fetchLogs();
  }, [fetchSandboxes, fetchLogs, activeTab]);

  // Create Sandbox
  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!newName || !newOwner) return;
    setIsSubmitting(true);
    try {
      const res = await fetch(`${API_BASE}/sandbox`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ sandbox_name: newName, owner: newOwner })
      });
      if (res.ok) {
        setNewName('');
        setNewOwner('');
        setActiveTab('sandboxes');
        fetchSandboxes();
      }
    } catch (err) {
      console.error(err);
    } finally {
      setIsSubmitting(false);
    }
  };

  // Delete Sandbox
  const handleDelete = async (id: number) => {
    if (!confirm('Are you sure you want to delete this sandbox environment?')) return;
    try {
      const res = await fetch(`${API_BASE}/sandbox/${id}`, { method: 'DELETE' });
      if (res.ok) {
        if (selectedSandbox?.id === id) {
          setSelectedSandbox(null);
          setTerminalOutputs([]);
        }
        fetchSandboxes();
      }
    } catch (err) {
      console.error(err);
    }
  };

  // Connect Sandbox (Simulated VSI)
  const handleConnect = async (id: number) => {
    // Optimistic state toggle to pending
    setSandboxes(prev => prev.map(s => s.id === id ? { ...s, status: 'PENDING' } : s));
    try {
      const res = await fetch(`${API_BASE}/sandbox/${id}/connect`, { method: 'POST' });
      const data = await res.json();
      if (res.ok) {
        fetchSandboxes();
      } else {
        alert('VSI connection attempt failed: ' + (data.details || data.error));
        fetchSandboxes();
      }
    } catch (err) {
      console.error(err);
      fetchSandboxes();
    }
  };

  // Disconnect Sandbox
  const handleDisconnect = async (id: number) => {
    try {
      const res = await fetch(`${API_BASE}/sandbox/${id}/disconnect`, { method: 'POST' });
      if (res.ok) {
        fetchSandboxes();
      }
    } catch (err) {
      console.error(err);
    }
  };

  // Send Command to VSI
  const handleSendCommand = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!commandInput || !selectedSandbox) return;
    const cmd = commandInput;
    setCommandInput('');
    setTerminalOutputs(prev => [...prev, `$ ${cmd}`]);

    try {
      const res = await fetch(`${API_BASE}/sandbox/${selectedSandbox.id}/run-command`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ command: cmd })
      });
      const data = await res.json();
      if (res.ok) {
        setTerminalOutputs(prev => [...prev, data.output]);
      } else {
        setTerminalOutputs(prev => [...prev, `ERROR: ${data.details || data.error}`]);
      }
    } catch (err) {
      setTerminalOutputs(prev => [...prev, 'System connection network timeout. Please verify host state.']);
    }
  };

  // Failure Simulation triggers
  const handleToggleFailure = async (type: string, val: any) => {
    try {
      let endpoint = '';
      let body: any = {};
      switch (type) {
        case 'api-delay':
          endpoint = '/simulate/api-delay';
          body = { delay_ms: Number(val) };
          setSimApiDelay(Number(val));
          break;
        case 'db-delay':
          endpoint = '/simulate/db-delay';
          body = { delay_ms: Number(val) };
          setSimDbDelay(Number(val));
          break;
        case 'db-failure':
          endpoint = '/simulate/db-failure';
          body = { enable: Boolean(val) };
          setSimDbFailure(Boolean(val));
          break;
        case 'vsi-timeout':
          endpoint = '/simulate/vsi-timeout';
          body = { enable: Boolean(val) };
          setSimVsiTimeout(Boolean(val));
          break;
        case 'random-errors':
          endpoint = '/simulate/random-errors';
          body = { enable: Boolean(val) };
          setSimRandomErrors(Boolean(val));
          break;
        case 'high-cpu':
          endpoint = '/simulate/high-cpu';
          body = { enable: Boolean(val) };
          setSimHighCpu(Boolean(val));
          break;
        case 'high-memory':
          endpoint = '/simulate/high-memory';
          body = { megabytes: Number(val), enable: Number(val) > 0 };
          setSimHighMemory(Number(val));
          break;
      }

      await fetch(`${API_BASE}${endpoint}`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(body)
      });
    } catch (err) {
      console.error(err);
    }
  };

  // Render Live SVG Chart (zero dependencies, p95/api latency visualizer)
  const renderSVGChart = () => {
    const maxVal = Math.max(...apiLatencyHistory, 50);
    const height = 150;
    const width = 500;
    const padding = 10;
    const step = (width - padding * 2) / (apiLatencyHistory.length - 1);
    
    // Build coordinate string
    const points = apiLatencyHistory.map((val, idx) => {
      const x = padding + idx * step;
      const y = height - padding - ((val / maxVal) * (height - padding * 2));
      return { x, y, val };
    });

    const pathD = points.reduce((acc, p, i) => {
      return i === 0 ? `M ${p.x} ${p.y}` : `${acc} L ${p.x} ${p.y}`;
    }, '');

    // Area fill
    const areaD = points.length > 0 
      ? `${pathD} L ${points[points.length - 1].x} ${height - padding} L ${points[0].x} ${height - padding} Z`
      : '';

    return (
      <svg viewBox={`0 0 ${width} ${height}`} style={{ width: '100%', height: '100%' }}>
        <defs>
          <linearGradient id="areaGrad" x1="0" y1="0" x2="0" y2="1">
            <stop offset="0%" stopColor="#8b5cf6" stopOpacity="0.4" />
            <stop offset="100%" stopColor="#8b5cf6" stopOpacity="0" />
          </linearGradient>
        </defs>
        
        {/* Horizontal grid lines */}
        {[0, 0.25, 0.5, 0.75, 1].map((ratio, i) => {
          const y = padding + ratio * (height - padding * 2);
          const gridVal = Math.round(maxVal - ratio * maxVal);
          return (
            <g key={i}>
              <line x1={padding} y1={y} x2={width - padding} y2={y} stroke="rgba(255,255,255,0.05)" strokeDasharray="3" />
              <text x={padding + 5} y={y - 3} fill="var(--text-muted)" fontSize="8" fontFamily="var(--font-mono)">
                {gridVal}ms
              </text>
            </g>
          );
        })}

        {/* Areas & Lines */}
        <path d={areaD} fill="url(#areaGrad)" />
        <path d={pathD} fill="none" stroke="var(--primary)" strokeWidth="2.5" />

        {/* Data point dot (last item) */}
        {points.length > 0 && (
          <circle 
            cx={points[points.length - 1].x} 
            cy={points[points.length - 1].y} 
            r="4" 
            fill="var(--secondary)" 
            style={{ filter: 'drop-shadow(0 0 5px var(--secondary))' }}
          />
        )}
      </svg>
    );
  };

  const getStatusClass = (status: string) => {
    switch (status.toUpperCase()) {
      case 'RUNNING': return 'indicator-running';
      case 'STOPPED': return 'indicator-stopped';
      case 'PENDING': return 'indicator-pending';
      case 'ERROR': return 'indicator-error';
      default: return 'indicator-stopped';
    }
  };

  return (
    <div style={{ display: 'flex', minHeight: '100vh' }}>
      
      {/* 1. Sidebar Navigation */}
      <aside style={{ width: '280px', backgroundColor: 'var(--bg-sidebar)', borderRight: '1px solid var(--border-color)', display: 'flex', flexDirection: 'column' }}>
        <div style={{ padding: '24px', borderBottom: '1px solid var(--border-color)', display: 'flex', alignItems: 'center', gap: '12px' }}>
          <Activity size={28} color="var(--secondary)" style={{ animation: 'pulseGlow 2s infinite' }} />
          <div>
            <h1 style={{ fontSize: '18px', fontWeight: '800', lineHeight: 1.2 }}>Sandbox Platform</h1>
            <span style={{ fontSize: '11px', color: 'var(--text-muted)', textTransform: 'uppercase', letterSpacing: '0.05em' }}>Observability Suite</span>
          </div>
        </div>

        <nav style={{ padding: '16px', display: 'flex', flexDirection: 'column', gap: '8px', flex: 1 }}>
          <button 
            className={`btn btn-secondary ${activeTab === 'dashboard' ? 'btn-primary' : ''}`}
            onClick={() => { setActiveTab('dashboard'); setSelectedSandbox(null); }}
            style={{ width: '100%', justifyContent: 'flex-start' }}
          >
            <Activity size={18} /> Dashboard Overview
          </button>
          
          <button 
            className={`btn btn-secondary ${activeTab === 'sandboxes' ? 'btn-primary' : ''}`}
            onClick={() => { setActiveTab('sandboxes'); }}
            style={{ width: '100%', justifyContent: 'flex-start' }}
          >
            <Layers size={18} /> Sandbox Environments
          </button>

          <button 
            className={`btn btn-secondary ${activeTab === 'create' ? 'btn-primary' : ''}`}
            onClick={() => { setActiveTab('create'); }}
            style={{ width: '100%', justifyContent: 'flex-start' }}
          >
            <PlusCircle size={18} /> Register Sandbox
          </button>

          <button 
            className={`btn btn-secondary ${activeTab === 'logs' ? 'btn-primary' : ''}`}
            onClick={() => { setActiveTab('logs'); }}
            style={{ width: '100%', justifyContent: 'flex-start' }}
          >
            <Terminal size={18} /> Sandbox Logs
          </button>

          <button 
            className={`btn btn-secondary ${activeTab === 'failures' ? 'btn-primary' : ''}`}
            onClick={() => { setActiveTab('failures'); }}
            style={{ width: '100%', justifyContent: 'flex-start', color: simApiDelay > 0 || simDbFailure || simVsiTimeout || simRandomErrors || simHighCpu || simHighMemory > 0 ? 'var(--warning)' : 'var(--text-primary)' }}
          >
            <Settings size={18} /> Failure Simulation
          </button>
        </nav>

        {/* Infrastructure health status */}
        <div style={{ padding: '20px', borderTop: '1px solid var(--border-color)', display: 'flex', flexDirection: 'column', gap: '12px', fontSize: '13px' }}>
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
            <span style={{ color: 'var(--text-secondary)' }}>System API Status</span>
            <div style={{ display: 'flex', alignItems: 'center', gap: '6px' }}>
              <span className={`indicator-pulse ${backendHealth === 'UP' ? 'indicator-running' : backendHealth === 'DOWN' ? 'indicator-error' : 'indicator-pending'}`} />
              <span style={{ fontWeight: 600, color: backendHealth === 'UP' ? 'var(--success)' : backendHealth === 'DOWN' ? 'var(--danger)' : 'var(--warning)' }}>
                {backendHealth}
              </span>
            </div>
          </div>
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
            <span style={{ color: 'var(--text-secondary)' }}>Live Ping Latency</span>
            <span style={{ fontFamily: 'var(--font-mono)', fontWeight: 600, color: 'var(--secondary)' }}>
              {apiLatencyHistory[apiLatencyHistory.length - 1]} ms
            </span>
          </div>
          <button 
            onClick={() => { fetchSandboxes(); fetchLogs(); }} 
            className="btn btn-secondary" 
            style={{ width: '100%', fontSize: '11px', padding: '6px 12px', justifyContent: 'center' }}
          >
            <RefreshCw size={12} /> Sync Telemetry
          </button>
        </div>
      </aside>

      {/* 2. Main Content Area */}
      <main style={{ flex: 1, padding: '40px', overflowY: 'auto' }}>
        
        {/* VIEW 1: DASHBOARD */}
        {activeTab === 'dashboard' && (
          <div style={{ display: 'flex', flexDirection: 'column', gap: '32px' }}>
            <div>
              <h2 style={{ fontSize: '28px', marginBottom: '8px' }}>Dashboard Overview</h2>
              <p style={{ color: 'var(--text-secondary)' }}>Real-time cluster health and virtual server connectivity status.</p>
            </div>

            {/* Micro KPI widgets */}
            <div className="grid-cols-4">
              <div className="glass-card" style={{ display: 'flex', alignItems: 'center', gap: '16px' }}>
                <div style={{ backgroundColor: 'rgba(139, 92, 246, 0.1)', padding: '12px', borderRadius: '12px', color: 'var(--primary)' }}>
                  <Layers size={24} />
                </div>
                <div>
                  <div style={{ fontSize: '24px', fontWeight: 700 }}>{sandboxes.length}</div>
                  <div style={{ fontSize: '12px', color: 'var(--text-secondary)' }}>Total Sandboxes</div>
                </div>
              </div>

              <div className="glass-card" style={{ display: 'flex', alignItems: 'center', gap: '16px' }}>
                <div style={{ backgroundColor: 'rgba(16, 185, 129, 0.1)', padding: '12px', borderRadius: '12px', color: 'var(--success)' }}>
                  <Server size={24} />
                </div>
                <div>
                  <div style={{ fontSize: '24px', fontWeight: 700 }}>
                    {sandboxes.filter(s => s.status === 'RUNNING').length}
                  </div>
                  <div style={{ fontSize: '12px', color: 'var(--text-secondary)' }}>Active VSIs</div>
                </div>
              </div>

              <div className="glass-card" style={{ display: 'flex', alignItems: 'center', gap: '16px' }}>
                <div style={{ backgroundColor: 'rgba(6, 182, 212, 0.1)', padding: '12px', borderRadius: '12px', color: 'var(--secondary)' }}>
                  <Database size={24} />
                </div>
                <div>
                  <div style={{ fontSize: '24px', fontWeight: 700 }}>
                    {backendHealth === 'UP' ? 'Connected' : 'Disconnected'}
                  </div>
                  <div style={{ fontSize: '12px', color: 'var(--text-secondary)' }}>Database Connection</div>
                </div>
              </div>

              <div className="glass-card" style={{ display: 'flex', alignItems: 'center', gap: '16px' }}>
                <div style={{ backgroundColor: 'rgba(245, 158, 11, 0.1)', padding: '12px', borderRadius: '12px', color: 'var(--warning)' }}>
                  <AlertTriangle size={24} />
                </div>
                <div>
                  <div style={{ fontSize: '24px', fontWeight: 700 }}>
                    {dbLogs.filter(l => l.log_level === 'ERROR').length}
                  </div>
                  <div style={{ fontSize: '12px', color: 'var(--text-secondary)' }}>Database Errors</div>
                </div>
              </div>
            </div>

            {/* Graphs Grid */}
            <div className="grid-cols-2">
              <div className="glass-card">
                <h3 style={{ fontSize: '16px', marginBottom: '16px', display: 'flex', alignItems: 'center', gap: '8px' }}>
                  <Activity size={18} color="var(--primary)" /> API Gateway Latency (Ping)
                </h3>
                <div style={{ height: '150px' }}>
                  {renderSVGChart()}
                </div>
              </div>

              <div className="glass-card" style={{ display: 'flex', flexDirection: 'column', justifyContent: 'center' }}>
                <h3 style={{ fontSize: '16px', marginBottom: '16px', display: 'flex', alignItems: 'center', gap: '8px' }}>
                  <Cpu size={18} color="var(--secondary)" /> Grafana Monitor
                </h3>
                <p style={{ fontSize: '13px', color: 'var(--text-secondary)', marginBottom: '20px' }}>
                  Open Grafana dashboards to analyze database query operations, container logs, system resource usage, and Golden Signals.
                </p>
                <div style={{ display: 'flex', gap: '12px' }}>
                  <a href="http://localhost:3001" target="_blank" rel="noreferrer" className="btn btn-primary">
                    Open Grafana Console
                  </a>
                </div>
              </div>
            </div>

            {/* Recent DB Logs */}
            <div className="glass-card">
              <h3 style={{ fontSize: '16px', marginBottom: '16px', display: 'flex', alignItems: 'center', gap: '8px' }}>
                <List size={18} color="var(--secondary)" /> Recent Database Event Logs
              </h3>
              <div style={{ overflowX: 'auto' }}>
                <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: '13px', textAlign: 'left' }}>
                  <thead>
                    <tr style={{ borderBottom: '1px solid var(--border-color)', color: 'var(--text-secondary)' }}>
                      <th style={{ padding: '12px' }}>Timestamp</th>
                      <th style={{ padding: '12px' }}>Sandbox ID</th>
                      <th style={{ padding: '12px' }}>Message</th>
                      <th style={{ padding: '12px' }}>Level</th>
                    </tr>
                  </thead>
                  <tbody>
                    {dbLogs.slice(0, 5).map(log => (
                      <tr key={log.id} style={{ borderBottom: '1px solid rgba(255,255,255,0.03)' }}>
                        <td style={{ padding: '12px', fontFamily: 'var(--font-mono)', color: 'var(--text-muted)' }}>
                          {new Date(log.created_at).toLocaleTimeString()}
                        </td>
                        <td style={{ padding: '12px', fontWeight: 600 }}>Sandbox-{log.sandbox_id}</td>
                        <td style={{ padding: '12px' }}>{log.message}</td>
                        <td style={{ padding: '12px' }}>
                          <span style={{ 
                            color: log.log_level === 'ERROR' ? 'var(--danger)' : log.log_level === 'WARNING' ? 'var(--warning)' : 'var(--success)',
                            fontWeight: 600
                          }}>
                            {log.log_level}
                          </span>
                        </td>
                      </tr>
                    ))}
                    {dbLogs.length === 0 && (
                      <tr>
                        <td colSpan={4} style={{ padding: '24px', textAlign: 'center', color: 'var(--text-muted)' }}>
                          No database log traces found. Provision and connect environments to populate database tables.
                        </td>
                      </tr>
                    )}
                  </tbody>
                </table>
              </div>
            </div>
          </div>
        )}

        {/* VIEW 2: SANDBOX LIST */}
        {activeTab === 'sandboxes' && (
          <div style={{ display: 'flex', flexDirection: 'column', gap: '32px' }}>
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
              <div>
                <h2 style={{ fontSize: '28px', marginBottom: '8px' }}>Sandbox Environments</h2>
                <p style={{ color: 'var(--text-secondary)' }}>Manage connection states, disconnect servers, and view terminals.</p>
              </div>
              <button onClick={() => { setActiveTab('create'); }} className="btn btn-primary">
                <PlusCircle size={16} /> New Sandbox
              </button>
            </div>

            <div className="grid-cols-2">
              {/* Left: Sandboxes list */}
              <div style={{ display: 'flex', flexDirection: 'column', gap: '16px' }}>
                {sandboxes.map(sandbox => (
                  <div 
                    key={sandbox.id} 
                    className="glass-card" 
                    style={{ 
                      borderLeft: selectedSandbox?.id === sandbox.id ? '4px solid var(--primary)' : '1px solid var(--border-color)',
                      cursor: 'pointer' 
                    }}
                    onClick={() => {
                      setSelectedSandbox(sandbox);
                      setTerminalOutputs([`Connected to Sandbox-${sandbox.id} [${sandbox.sandbox_name}] console.\nType commands below to run on the VSI.`]);
                    }}
                  >
                    <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', marginBottom: '12px' }}>
                      <div>
                        <h3 style={{ fontSize: '18px' }}>{sandbox.sandbox_name}</h3>
                        <span style={{ fontSize: '12px', color: 'var(--text-muted)' }}>Owner: {sandbox.owner}</span>
                      </div>
                      <div style={{ display: 'flex', alignItems: 'center', gap: '6px', fontSize: '13px' }}>
                        <span className={`indicator-pulse ${getStatusClass(sandbox.status)}`} />
                        <span style={{ fontWeight: 600 }}>{sandbox.status}</span>
                      </div>
                    </div>

                    <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginTop: '16px', borderTop: '1px solid var(--border-color)', paddingTop: '16px' }}>
                      <div style={{ display: 'flex', gap: '8px' }}>
                        {sandbox.status !== 'RUNNING' && sandbox.status !== 'PENDING' && (
                          <button onClick={(e) => { e.stopPropagation(); handleConnect(sandbox.id); }} className="btn btn-success" style={{ padding: '6px 12px', fontSize: '12px' }}>
                            <Play size={12} /> Connect VSI
                          </button>
                        )}
                        {sandbox.status === 'RUNNING' && (
                          <button onClick={(e) => { e.stopPropagation(); handleDisconnect(sandbox.id); }} className="btn btn-secondary" style={{ padding: '6px 12px', fontSize: '12px', color: 'var(--danger)' }}>
                            <Power size={12} /> Disconnect
                          </button>
                        )}
                      </div>
                      
                      <button onClick={(e) => { e.stopPropagation(); handleDelete(sandbox.id); }} className="btn btn-secondary" style={{ padding: '6px 12px', fontSize: '12px', color: 'var(--text-muted)' }}>
                        <Trash2 size={12} /> Delete
                      </button>
                    </div>
                  </div>
                ))}

                {sandboxes.length === 0 && (
                  <div className="glass-card" style={{ textAlign: 'center', padding: '40px', color: 'var(--text-secondary)' }}>
                    No sandboxes registered. Use the register tab to provision one.
                  </div>
                )}
              </div>

              {/* Right: Terminal Console */}
              <div style={{ display: 'flex', flexDirection: 'column' }}>
                {selectedSandbox ? (
                  <div className="glass-card" style={{ flex: 1, display: 'flex', flexDirection: 'column', backgroundColor: 'var(--terminal-bg)', border: '1px solid rgba(255,255,255,0.1)', padding: '0', borderRadius: '16px', overflow: 'hidden', minHeight: '350px' }}>
                    <div style={{ backgroundColor: 'rgba(255,255,255,0.03)', padding: '12px 20px', borderBottom: '1px solid rgba(255,255,255,0.05)', display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                      <span style={{ fontSize: '13px', fontWeight: 600, fontFamily: 'var(--font-mono)', color: 'var(--text-secondary)' }}>
                        Terminal Shell: {selectedSandbox.sandbox_name}
                      </span>
                      <span style={{ fontSize: '11px', color: 'var(--success)', fontFamily: 'var(--font-mono)' }}>
                        Status: {selectedSandbox.status}
                      </span>
                    </div>

                    <div style={{ flex: 1, padding: '20px', overflowY: 'auto', fontFamily: 'var(--font-mono)', fontSize: '13px', color: '#10b981', display: 'flex', flexDirection: 'column', gap: '8px' }}>
                      {terminalOutputs.map((line, idx) => (
                        <pre key={idx} style={{ whiteSpace: 'pre-wrap', margin: 0 }}>{line}</pre>
                      ))}
                      <div ref={terminalEndRef} />
                    </div>

                    <form onSubmit={handleSendCommand} style={{ borderTop: '1px solid rgba(255,255,255,0.05)', padding: '12px 20px', display: 'flex', gap: '10px' }}>
                      <span style={{ fontFamily: 'var(--font-mono)', color: 'var(--text-secondary)', alignSelf: 'center' }}>$</span>
                      <input 
                        type="text" 
                        value={commandInput}
                        onChange={(e) => setCommandInput(e.target.value)}
                        placeholder={selectedSandbox.status === 'RUNNING' ? "Type command (e.g. uname -a, df -h, fail)..." : "VSI disconnected. Connect to run command."}
                        disabled={selectedSandbox.status !== 'RUNNING'}
                        style={{ flex: 1, backgroundColor: 'transparent', border: 'none', color: '#fff', fontFamily: 'var(--font-mono)', outline: 'none', fontSize: '13px' }}
                      />
                    </form>
                  </div>
                ) : (
                  <div className="glass-card" style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', justifyContent: 'center', textAlign: 'center', height: '100%', minHeight: '350px', color: 'var(--text-secondary)' }}>
                    <Terminal size={48} style={{ marginBottom: '16px', opacity: 0.3 }} />
                    <h3>No Terminal Active</h3>
                    <p style={{ maxWidth: '280px', fontSize: '13px', marginTop: '8px' }}>Select an environment from the list to spin up a terminal session.</p>
                  </div>
                )}
              </div>
            </div>
          </div>
        )}

        {/* VIEW 3: CREATE SANDBOX */}
        {activeTab === 'create' && (
          <div style={{ display: 'flex', flexDirection: 'column', gap: '32px', maxWidth: '600px', margin: '0 auto' }}>
            <div>
              <h2 style={{ fontSize: '28px', marginBottom: '8px' }}>Register Sandbox</h2>
              <p style={{ color: 'var(--text-secondary)' }}>Create a new sandbox management record in PostgreSQL.</p>
            </div>

            <div className="glass-card">
              <form onSubmit={handleCreate} style={{ display: 'flex', flexDirection: 'column', gap: '20px' }}>
                <div style={{ display: 'flex', flexDirection: 'column', gap: '8px' }}>
                  <label style={{ fontSize: '14px', fontWeight: 600, color: 'var(--text-secondary)' }}>Sandbox Name</label>
                  <input 
                    type="text" 
                    placeholder="e.g. dev-sandbox-vsi" 
                    value={newName}
                    onChange={(e) => setNewName(e.target.value)}
                    required
                    style={{ backgroundColor: 'rgba(255,255,255,0.03)', border: '1px solid var(--border-color)', borderRadius: '8px', padding: '12px', color: '#fff', fontSize: '14px', outline: 'none' }}
                  />
                </div>

                <div style={{ display: 'flex', flexDirection: 'column', gap: '8px' }}>
                  <label style={{ fontSize: '14px', fontWeight: 600, color: 'var(--text-secondary)' }}>Owner Email / ID</label>
                  <input 
                    type="email" 
                    placeholder="e.g. admin@enterprise.com" 
                    value={newOwner}
                    onChange={(e) => setNewOwner(e.target.value)}
                    required
                    style={{ backgroundColor: 'rgba(255,255,255,0.03)', border: '1px solid var(--border-color)', borderRadius: '8px', padding: '12px', color: '#fff', fontSize: '14px', outline: 'none' }}
                  />
                </div>

                <div style={{ marginTop: '10px' }}>
                  <button type="submit" disabled={isSubmitting} className="btn btn-primary" style={{ width: '100%', justifyContent: 'center' }}>
                    {isSubmitting ? 'Registering...' : 'Initialize Environment'}
                  </button>
                </div>
              </form>
            </div>
          </div>
        )}

        {/* VIEW 4: LOGS VIEW */}
        {activeTab === 'logs' && (
          <div style={{ display: 'flex', flexDirection: 'column', gap: '32px' }}>
            <div>
              <h2 style={{ fontSize: '28px', marginBottom: '8px' }}>Sandbox Logs Explorer</h2>
              <p style={{ color: 'var(--text-secondary)' }}>Query system audit traces stored in the database logs table.</p>
            </div>

            {/* Filters */}
            <div className="glass-card" style={{ display: 'flex', gap: '16px', alignItems: 'center', flexWrap: 'wrap' }}>
              <div style={{ display: 'flex', alignItems: 'center', gap: '8px', backgroundColor: 'rgba(255,255,255,0.03)', border: '1px solid var(--border-color)', borderRadius: '8px', padding: '8px 12px', flex: 1, minWidth: '200px' }}>
                <Search size={16} color="var(--text-muted)" />
                <input 
                  type="text" 
                  placeholder="Filter by Sandbox ID..." 
                  value={logFilterId}
                  onChange={(e) => setLogFilterId(e.target.value)}
                  style={{ backgroundColor: 'transparent', border: 'none', color: '#fff', outline: 'none', fontSize: '13px', width: '100%' }}
                />
              </div>

              <div style={{ display: 'flex', alignItems: 'center', gap: '8px', backgroundColor: 'rgba(255,255,255,0.03)', border: '1px solid var(--border-color)', borderRadius: '8px', padding: '8px 12px', minWidth: '150px' }}>
                <select 
                  value={logFilterLevel}
                  onChange={(e) => setLogFilterLevel(e.target.value)}
                  style={{ backgroundColor: 'transparent', border: 'none', color: '#fff', outline: 'none', fontSize: '13px', width: '100%', cursor: 'pointer' }}
                >
                  <option value="" style={{ background: '#0a0f1d' }}>All Log Levels</option>
                  <option value="INFO" style={{ background: '#0a0f1d' }}>INFO</option>
                  <option value="WARNING" style={{ background: '#0a0f1d' }}>WARNING</option>
                  <option value="ERROR" style={{ background: '#0a0f1d' }}>ERROR</option>
                </select>
              </div>

              <button onClick={fetchLogs} className="btn btn-secondary">
                Query Logs
              </button>
            </div>

            {/* Logs Output Console */}
            <div className="glass-card" style={{ backgroundColor: 'var(--terminal-bg)', padding: '0', overflow: 'hidden' }}>
              <div style={{ borderBottom: '1px solid var(--border-color)', padding: '12px 20px', display: 'flex', justifyContent: 'space-between', color: 'var(--text-secondary)', fontSize: '13px' }}>
                <span>SQL Database Logs View</span>
                <span>Recent Traces</span>
              </div>
              <div style={{ padding: '20px', maxHeight: '500px', overflowY: 'auto', display: 'flex', flexDirection: 'column', gap: '10px', fontFamily: 'var(--font-mono)', fontSize: '12px' }}>
                {dbLogs.map(log => (
                  <div key={log.id} style={{ display: 'flex', gap: '16px', borderBottom: '1px solid rgba(255,255,255,0.02)', paddingBottom: '8px' }}>
                    <span style={{ color: 'var(--text-muted)' }}>[{new Date(log.created_at).toISOString()}]</span>
                    <span style={{ color: 'var(--secondary)' }}>Sandbox-{log.sandbox_id}</span>
                    <span style={{ 
                      color: log.log_level === 'ERROR' ? 'var(--danger)' : log.log_level === 'WARNING' ? 'var(--warning)' : 'var(--success)',
                      fontWeight: 600
                    }}>[{log.log_level}]</span>
                    <span style={{ color: '#e2e8f0' }}>{log.message}</span>
                  </div>
                ))}

                {dbLogs.length === 0 && (
                  <div style={{ padding: '40px', textAlign: 'center', color: 'var(--text-muted)' }}>
                    No audit records match the selected queries.
                  </div>
                )}
              </div>
            </div>
          </div>
        )}

        {/* VIEW 5: FAILURE SIMULATION */}
        {activeTab === 'failures' && (
          <div style={{ display: 'flex', flexDirection: 'column', gap: '32px' }}>
            <div>
              <h2 style={{ fontSize: '28px', marginBottom: '8px', color: 'var(--warning)', display: 'flex', alignItems: 'center', gap: '12px' }}>
                <Settings size={28} /> Failure Simulation Console
              </h2>
              <p style={{ color: 'var(--text-secondary)' }}>Inject artificial latency and runtime failure conditions to test alerts and dashboard graphs.</p>
            </div>

            <div className="grid-cols-2">
              
              {/* Latency and Error controls */}
              <div style={{ display: 'flex', flexDirection: 'column', gap: '24px' }}>
                <div className="glass-card">
                  <h3 style={{ fontSize: '18px', marginBottom: '20px', display: 'flex', alignItems: 'center', gap: '8px' }}>
                    <Clock size={18} color="var(--primary)" /> API & Database Latency
                  </h3>
                  
                  <div style={{ display: 'flex', flexDirection: 'column', gap: '16px' }}>
                    <div>
                      <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: '8px', fontSize: '14px' }}>
                        <span>API Request delay:</span>
                        <span style={{ fontWeight: 600, color: 'var(--primary)' }}>{simApiDelay} ms</span>
                      </div>
                      <input 
                        type="range" 
                        min="0" 
                        max="5000" 
                        step="250"
                        value={simApiDelay}
                        onChange={(e) => handleToggleFailure('api-delay', e.target.value)}
                        style={{ width: '100%', accentColor: 'var(--primary)' }}
                      />
                    </div>

                    <div>
                      <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: '8px', fontSize: '14px' }}>
                        <span>GORM DB Query delay:</span>
                        <span style={{ fontWeight: 600, color: 'var(--primary)' }}>{simDbDelay} ms</span>
                      </div>
                      <input 
                        type="range" 
                        min="0" 
                        max="5000" 
                        step="250"
                        value={simDbDelay}
                        onChange={(e) => handleToggleFailure('db-delay', e.target.value)}
                        style={{ width: '100%', accentColor: 'var(--primary)' }}
                      />
                    </div>
                  </div>
                </div>

                <div className="glass-card">
                  <h3 style={{ fontSize: '18px', marginBottom: '20px', display: 'flex', alignItems: 'center', gap: '8px' }}>
                    <AlertTriangle size={18} color="var(--danger)" /> API & DB Error Injection
                  </h3>

                  <div style={{ display: 'flex', flexDirection: 'column', gap: '16px' }}>
                    <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                      <div>
                        <div style={{ fontWeight: 600 }}>Simulate SQL Operations failure</div>
                        <div style={{ fontSize: '12px', color: 'var(--text-secondary)' }}>Causes GORM queries to crash and trigger 500 status</div>
                      </div>
                      <button 
                        onClick={() => handleToggleFailure('db-failure', !simDbFailure)}
                        className={`btn ${simDbFailure ? 'btn-danger' : 'btn-secondary'}`}
                      >
                        {simDbFailure ? 'Enabled' : 'Disabled'}
                      </button>
                    </div>

                    <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', borderTop: '1px solid var(--border-color)', paddingTop: '16px' }}>
                      <div>
                        <div style={{ fontWeight: 600 }}>Simulate Random HTTP API Errors</div>
                        <div style={{ fontSize: '12px', color: 'var(--text-secondary)' }}>Injects random 500 status on 25% of REST calls</div>
                      </div>
                      <button 
                        onClick={() => handleToggleFailure('random-errors', !simRandomErrors)}
                        className={`btn ${simRandomErrors ? 'btn-danger' : 'btn-secondary'}`}
                      >
                        {simRandomErrors ? 'Enabled' : 'Disabled'}
                      </button>
                    </div>
                  </div>
                </div>
              </div>

              {/* Resource Saturation controls */}
              <div style={{ display: 'flex', flexDirection: 'column', gap: '24px' }}>
                <div className="glass-card">
                  <h3 style={{ fontSize: '18px', marginBottom: '20px', display: 'flex', alignItems: 'center', gap: '8px' }}>
                    <Cpu size={18} color="var(--secondary)" /> Resource Saturation
                  </h3>

                  <div style={{ display: 'flex', flexDirection: 'column', gap: '16px' }}>
                    <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                      <div>
                        <div style={{ fontWeight: 600 }}>CPU Burn Threads</div>
                        <div style={{ fontSize: '12px', color: 'var(--text-secondary)' }}>Spawns 4 worker threads running infinite calculations</div>
                      </div>
                      <button 
                        onClick={() => handleToggleFailure('high-cpu', !simHighCpu)}
                        className={`btn ${simHighCpu ? 'btn-danger' : 'btn-secondary'}`}
                      >
                        {simHighCpu ? 'Enabled' : 'Disabled'}
                      </button>
                    </div>

                    <div style={{ borderTop: '1px solid var(--border-color)', paddingTop: '16px' }}>
                      <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: '8px', fontSize: '14px' }}>
                        <span>Trigger Memory leak (MB):</span>
                        <span style={{ fontWeight: 600, color: 'var(--secondary)' }}>{simHighMemory} MB</span>
                      </div>
                      <div style={{ display: 'flex', gap: '10px' }}>
                        <button onClick={() => handleToggleFailure('high-memory', 50)} className="btn btn-secondary" style={{ flex: 1 }}>+50 MB</button>
                        <button onClick={() => handleToggleFailure('high-memory', 100)} className="btn btn-secondary" style={{ flex: 1 }}>+100 MB</button>
                        <button onClick={() => handleToggleFailure('high-memory', 0)} className="btn btn-danger" style={{ flex: 1 }}>Release Leak</button>
                      </div>
                      <p style={{ fontSize: '11px', color: 'var(--text-muted)', marginTop: '8px' }}>
                        Appends allocated bytes block inside memory leak pool to trigger RAM usage graphs spikes.
                      </p>
                    </div>
                  </div>
                </div>

                <div className="glass-card">
                  <h3 style={{ fontSize: '18px', marginBottom: '20px', display: 'flex', alignItems: 'center', gap: '8px' }}>
                    <Server size={18} color="var(--warning)" /> VSI Connectivity Timeout
                  </h3>

                  <div style={{ display: 'flex', flexDirection: 'column', gap: '16px' }}>
                    <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                      <div>
                        <div style={{ fontWeight: 600 }}>Force VSI Connection Timeout</div>
                        <div style={{ fontSize: '12px', color: 'var(--text-secondary)' }}>Causes VSI connection attempts to hang and fail</div>
                      </div>
                      <button 
                        onClick={() => handleToggleFailure('vsi-timeout', !simVsiTimeout)}
                        className={`btn ${simVsiTimeout ? 'btn-danger' : 'btn-secondary'}`}
                      >
                        {simVsiTimeout ? 'Enabled' : 'Disabled'}
                      </button>
                    </div>
                  </div>
                </div>
              </div>

            </div>
          </div>
        )}

      </main>
    </div>
  );
}
