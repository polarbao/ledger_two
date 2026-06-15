import React, { useState, useEffect } from 'react';
import { useLedgerStore } from '../../stores/ledger.store';
import { ledgerApi, LedgerMember } from '../../api/ledger.api';
import { Users, UserPlus, Shield, X, Edit2 } from 'lucide-react';
import { ApiError } from '../../api/client';

export default function LedgerSettings() {
  const { activeLedgerId, activeRole } = useLedgerStore();
  const [members, setMembers] = useState<LedgerMember[]>([]);
  const [loading, setLoading] = useState(false);
  const [errorMsg, setErrorMsg] = useState<string | null>(null);

  const [inviteUsername, setInviteUsername] = useState('');
  const [inviteRole, setInviteRole] = useState('viewer');
  
  const [newLedgerName, setNewLedgerName] = useState('');
  const [creatingLedger, setCreatingLedger] = useState(false);

  const fetchMembers = async () => {
    if (!activeLedgerId) return;
    setLoading(true);
    setErrorMsg(null);
    try {
      const data = await ledgerApi.getLedgerMembers(activeLedgerId);
      setMembers(data);
    } catch (err: unknown) {
      if (err instanceof ApiError) {
        setErrorMsg(err.message);
      }
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchMembers();
  }, [activeLedgerId]);

  const handleInvite = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!activeLedgerId || !inviteUsername) return;
    setErrorMsg(null);
    try {
      await ledgerApi.addMember(activeLedgerId, { username: inviteUsername, role: inviteRole });
      setInviteUsername('');
      fetchMembers();
    } catch (err: unknown) {
      if (err instanceof ApiError) {
        setErrorMsg(err.message);
      }
    }
  };

  const handleUpdateRole = async (userId: string, newRole: string) => {
    if (!activeLedgerId) return;
    setErrorMsg(null);
    try {
      await ledgerApi.updateMemberRole(activeLedgerId, userId, { role: newRole });
      fetchMembers();
    } catch (err: unknown) {
      if (err instanceof ApiError) {
        setErrorMsg(err.message);
      }
    }
  };

  const handleRemove = async (userId: string) => {
    if (!activeLedgerId) return;
    if (!window.confirm('确定要移除该成员吗？')) return;
    setErrorMsg(null);
    try {
      await ledgerApi.removeMember(activeLedgerId, userId);
      fetchMembers();
    } catch (err: unknown) {
      if (err instanceof ApiError) {
        setErrorMsg(err.message);
      }
    }
  };

  const handleCreateLedger = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!newLedgerName) return;
    setCreatingLedger(true);
    setErrorMsg(null);
    try {
      const res = await ledgerApi.createLedger({ name: newLedgerName });
      window.location.reload(); // Reload to refresh sidebar ledgers and set new active?
    } catch (err: unknown) {
      if (err instanceof ApiError) {
        setErrorMsg(err.message);
      }
    } finally {
      setCreatingLedger(false);
    }
  };

  if (activeRole !== 'owner') {
    return (
      <div className="glass-card" style={{ padding: '20px', marginTop: '20px' }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: '8px', color: 'var(--text-muted)' }}>
          <Shield size={18} />
          <span>您当前的角色是 <strong>{activeRole}</strong>，无法管理账本成员。只有 Owner 具有管理权限。</span>
        </div>
      </div>
    );
  }

  return (
    <div className="glass-card" style={{ padding: '20px', marginTop: '20px', display: 'flex', flexDirection: 'column', gap: '20px' }}>
      <div style={{ display: 'flex', alignItems: 'center', gap: '10px', borderBottom: '1px solid rgba(255, 255, 255, 0.05)', paddingBottom: '12px' }}>
        <Users size={20} className="partner-highlight" />
        <h3 style={{ margin: 0, fontSize: '16px', fontWeight: 600 }}>账本成员管理</h3>
      </div>

      {errorMsg && (
        <div className="error-banner" style={{ padding: '10px', borderRadius: '8px' }}>
          {errorMsg}
        </div>
      )}

      {/* Create Ledger Form */}
      <form onSubmit={handleCreateLedger} style={{ display: 'flex', gap: '10px', alignItems: 'center', marginBottom: '10px' }}>
        <input 
          type="text" 
          placeholder="输入新账本名称" 
          value={newLedgerName}
          onChange={e => setNewLedgerName(e.target.value)}
          className="form-input"
          style={{ flexGrow: 1 }}
          required
        />
        <button type="submit" className="btn-primary" style={{ padding: '10px 16px' }} disabled={creatingLedger}>
          创建新账本
        </button>
      </form>

      <div style={{ height: '1px', background: 'rgba(255,255,255,0.05)', margin: '10px 0' }} />

      {/* Invite Form */}
      <form onSubmit={handleInvite} style={{ display: 'flex', gap: '10px', alignItems: 'center' }}>
        <input 
          type="text" 
          placeholder="输入要邀请的用户名" 
          value={inviteUsername}
          onChange={e => setInviteUsername(e.target.value)}
          className="form-input"
          style={{ flexGrow: 1 }}
          required
        />
        <select 
          value={inviteRole} 
          onChange={e => setInviteRole(e.target.value)}
          className="form-input"
          style={{ width: '120px' }}
        >
          <option value="editor">Editor</option>
          <option value="viewer">Viewer</option>
        </select>
        <button type="submit" className="btn-primary" style={{ padding: '10px 16px', display: 'flex', gap: '6px' }}>
          <UserPlus size={16} /> 添加成员
        </button>
      </form>

      {/* Member List */}
      <div style={{ display: 'flex', flexDirection: 'column', gap: '10px' }}>
        {loading && <div style={{ color: 'var(--text-muted)' }}>加载中...</div>}
        {!loading && members.map(m => (
          <div key={m.user_id} style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', background: 'rgba(255,255,255,0.02)', padding: '12px', borderRadius: '8px' }}>
            <div>
              <span style={{ fontWeight: 600 }}>{m.username}</span>
              <span style={{ marginLeft: '10px', fontSize: '12px', background: 'rgba(255,255,255,0.1)', padding: '2px 6px', borderRadius: '4px' }}>{m.role}</span>
            </div>
            
            {m.role !== 'owner' && (
              <div style={{ display: 'flex', gap: '10px' }}>
                <select 
                  value={m.role}
                  onChange={(e) => handleUpdateRole(m.user_id, e.target.value)}
                  className="form-input"
                  style={{ padding: '4px 8px', fontSize: '12px', height: 'auto' }}
                >
                  <option value="editor">Editor</option>
                  <option value="viewer">Viewer</option>
                </select>
                <button onClick={() => handleRemove(m.user_id)} className="btn-danger" style={{ padding: '4px 8px', fontSize: '12px' }}>
                  <X size={14} /> 移除
                </button>
              </div>
            )}
          </div>
        ))}
      </div>
    </div>
  );
}
