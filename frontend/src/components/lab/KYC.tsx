import React, { useState } from 'react';
import { UserCheck, UserPlus, List, Loader2, AlertTriangle, Shield, XCircle } from 'lucide-react';
import {
  checkKycStatus,
  grantKycAccess,
  getKycWhitelist,
  KYCStatusRequest,
  KYCGrantRequest,
} from '../../lib/api';
import { toast } from '../ui/toast';

const KYC: React.FC = () => {
  // Check KYC Status State
  const [checkUserId, setCheckUserId] = useState('');
  const [kycStatusResult, setKycStatusResult] = useState<any>(null);
  const [isCheckingKyc, setIsCheckingKyc] = useState(false);
  const [checkKycError, setCheckKycError] = useState<string | null>(null);

  // Grant KYC Access State
  const [grantUserId, setGrantUserId] = useState('');
  const [grantReason, setGrantReason] = useState('');
  const [isGrantingKyc, setIsGrantingKyc] = useState(false);
  const [grantKycError, setGrantKycError] = useState<string | null>(null);

  // View Whitelist State
  const [whitelistUser, setWhitelistUser] = useState(''); // Can be specific user or 'all' (if API supports)
  const [whitelistResult, setWhitelistResult] = useState<string[] | null>(null);
  const [isViewingWhitelist, setIsViewingWhitelist] = useState(false);
  const [whitelistError, setWhitelistError] = useState<string | null>(null);

  const handleCheckKycStatus = async (e: React.FormEvent) => {
    e.preventDefault();
    setIsCheckingKyc(true);
    setKycStatusResult(null);
    setCheckKycError(null);
    try {
      const request: KYCStatusRequest = { userId: checkUserId };
      const res = await checkKycStatus(request);
      if (res.success && res.data) {
        setKycStatusResult(res.data);
        toast.success(`KYC status for ${checkUserId} fetched.`);
      } else {
        setCheckKycError(res.error || 'Failed to check KYC status.');
        toast.error(res.error || 'Failed to check KYC status.');
      }
    } catch (err) {
      setCheckKycError('An unexpected error occurred.');
      toast.error('An unexpected error occurred.');
      console.error(err);
    } finally {
      setIsCheckingKyc(false);
    }
  };

  const handleGrantKycAccess = async (e: React.FormEvent) => {
    e.preventDefault();
    setIsGrantingKyc(true);
    setGrantKycError(null);
    try {
      const request: KYCGrantRequest = { userId: grantUserId, reason: grantReason || undefined };
      const res = await grantKycAccess(request);
      if (res.success && res.data) {
        toast.success(`KYC access granted to ${grantUserId}!`);
        setGrantUserId('');
        setGrantReason('');
        // Optionally refresh whitelist if it's currently displayed
        if (whitelistUser === 'all' || whitelistUser === grantUserId) {
          handleViewWhitelist({ preventDefault: () => {} } as React.FormEvent);
        }
      } else {
        setGrantKycError(res.error || 'Failed to grant KYC access.');
        toast.error(res.error || 'Failed to grant KYC access.');
      }
    } catch (err) {
      setGrantKycError('An unexpected error occurred.');
      toast.error('An unexpected error occurred.');
      console.error(err);
    } finally {
      setIsGrantingKyc(false);
    }
  };

  const handleViewWhitelist = async (e: React.FormEvent) => {
    e.preventDefault();
    setIsViewingWhitelist(true);
    setWhitelistResult(null);
    setWhitelistError(null);
    try {
      const res = await getKycWhitelist(whitelistUser || 'all'); // Assume 'all' if input is empty
      if (res.success && res.data) {
        setWhitelistResult(res.data.users);
        toast.success('KYC whitelist fetched.');
      } else {
        setWhitelistError(res.error || 'Failed to fetch KYC whitelist.');
        toast.error(res.error || 'Failed to fetch KYC whitelist.');
      }
    } catch (err) {
      setWhitelistError('An unexpected error occurred.');
      toast.error('An unexpected error occurred.');
      console.error(err);
    } finally {
      setIsViewingWhitelist(false);
    }
  };

  return (
    <div className="p-4">
      <h2 className="text-2xl font-bold text-gray-100 mb-6">KYC Operations Lab</h2>
      <p className="text-gray-400 mb-6">
        Manage and verify Know Your Customer (KYC) statuses for users, ensuring privacy-preserving compliance on Casper.
      </p>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Check KYC Status */}
        <div className="bg-[#1a1a2a] p-6 rounded-lg border border-[#222235] shadow-md">
          <h3 className="text-xl font-semibold text-gray-100 mb-4 flex items-center gap-2">
            <UserCheck size={24} className="text-red-500" />
            Check KYC Status
          </h3>
          <form onSubmit={handleCheckKycStatus} className="space-y-4">
            <div>
              <label htmlFor="checkUserId" className="block text-sm font-medium text-gray-300 mb-1">
                User ID
              </label>
              <input
                type="text"
                id="checkUserId"
                value={checkUserId}
                onChange={(e) => setCheckUserId(e.target.value)}
                className="w-full p-2 bg-[#0b0b10] border border-[#222235] rounded-md text-gray-100 focus:ring-red-500 focus:border-red-500"
                placeholder="Enter User ID"
                required
              />
            </div>
            <button
              type="submit"
              disabled={isCheckingKyc}
              className="w-full flex items-center justify-center gap-2 px-4 py-2 bg-red-600 hover:bg-red-700 text-white rounded-md transition-colors duration-200 disabled:opacity-50 disabled:cursor-not-allowed"
            >
              {isCheckingKyc ? <Loader2 size={20} className="animate-spin" /> : <UserCheck size={20} />}
              {isCheckingKyc ? 'Checking...' : 'Check Status'}
            </button>
          </form>

          {checkKycError && (
            <div className="mt-4 p-3 bg-red-900/30 text-red-300 border border-red-700 rounded-md flex items-center gap-2">
              <AlertTriangle size={20} /> {checkKycError}
            </div>
          )}

          {kycStatusResult && (
            <div className="mt-6 p-4 bg-[#0b0b10] border border-[#222235] rounded-md space-y-2 text-sm">
              <h4 className="text-lg font-semibold text-red-400">KYC Status for {kycStatusResult.userId}:</h4>
              <p><span className="font-medium text-gray-300">Whitelisted:</span> <span className={`font-bold ${kycStatusResult.isWhitelisted ? 'text-green-400' : 'text-red-400'}`}>{kycStatusResult.isWhitelisted ? 'YES' : 'NO'}</span></p>
              <p><span className="font-medium text-gray-300">Status:</span> <span className="font-mono break-all">{kycStatusResult.status}</span></p>
              <p className="text-gray-500 mt-2">
                <Shield size={16} className="inline-block mr-1" />
                Privacy Note: Only whitelisted status and general status are revealed. No personal data is exposed.
              </p>
            </div>
          )}
        </div>

        {/* Grant KYC Access */}
        <div className="bg-[#1a1a2a] p-6 rounded-lg border border-[#222235] shadow-md">
          <h3 className="text-xl font-semibold text-gray-100 mb-4 flex items-center gap-2">
            <UserPlus size={24} className="text-red-500" />
            Grant KYC Access
          </h3>
          <form onSubmit={handleGrantKycAccess} className="space-y-4">
            <div>
              <label htmlFor="grantUserId" className="block text-sm font-medium text-gray-300 mb-1">
                User ID to Grant
              </label>
              <input
                type="text"
                id="grantUserId"
                value={grantUserId}
                onChange={(e) => setGrantUserId(e.target.value)}
                className="w-full p-2 bg-[#0b0b10] border border-[#222235] rounded-md text-gray-100 focus:ring-red-500 focus:border-red-500"
                placeholder="Enter User ID"
                required
              />
            </div>
            <div>
              <label htmlFor="grantReason" className="block text-sm font-medium text-gray-300 mb-1">
                Reason (Optional)
              </label>
              <textarea
                id="grantReason"
                rows={3}
                value={grantReason}
                onChange={(e) => setGrantReason(e.target.value)}
                className="w-full p-2 bg-[#0b0b10] border border-[#222235] rounded-md text-gray-100 focus:ring-red-500 focus:border-red-500"
                placeholder="e.g., 'Manual verification after document submission'"
              ></textarea>
            </div>
            <button
              type="submit"
              disabled={isGrantingKyc}
              className="w-full flex items-center justify-center gap-2 px-4 py-2 bg-red-600 hover:bg-red-700 text-white rounded-md transition-colors duration-200 disabled:opacity-50 disabled:cursor-not-allowed"
            >
              {isGrantingKyc ? <Loader2 size={20} className="animate-spin" /> : <UserPlus size={20} />}
              {isGrantingKyc ? 'Granting...' : 'Grant Access'}
            </button>
          </form>

          {grantKycError && (
            <div className="mt-4 p-3 bg-red-900/30 text-red-300 border border-red-700 rounded-md flex items-center gap-2">
              <AlertTriangle size={20} /> {grantKycError}
            </div>
          )}
        </div>
      </div>

      {/* View Whitelist */}
      <div className="mt-8 bg-[#1a1a2a] p-6 rounded-lg border border-[#222235] shadow-md">
        <h3 className="text-xl font-semibold text-gray-100 mb-4 flex items-center gap-2">
          <List size={24} className="text-red-500" />
          View KYC Whitelist
        </h3>
        <form onSubmit={handleViewWhitelist} className="space-y-4">
          <div>
            <label htmlFor="whitelistUser" className="block text-sm font-medium text-gray-300 mb-1">
              Specific User ID (Optional, leave empty for all)
            </label>
            <input
              type="text"
              id="whitelistUser"
              value={whitelistUser}
              onChange={(e) => setWhitelistUser(e.target.value)}
              className="w-full p-2 bg-[#0b0b10] border border-[#222235] rounded-md text-gray-100 focus:ring-red-500 focus:border-red-500"
              placeholder="Enter User ID or leave empty for all"
            />
          </div>
          <button
            type="submit"
            disabled={isViewingWhitelist}
            className="w-full flex items-center justify-center gap-2 px-4 py-2 bg-red-600 hover:bg-red-700 text-white rounded-md transition-colors duration-200 disabled:opacity-50 disabled:cursor-not-allowed"
          >
            {isViewingWhitelist ? <Loader2 size={20} className="animate-spin" /> : <List size={20} />}
            {isViewingWhitelist ? 'Loading Whitelist...' : 'View Whitelist'}
          </button>
        </form>

        {whitelistError && (
          <div className="mt-4 p-3 bg-red-900/30 text-red-300 border border-red-700 rounded-md flex items-center gap-2">
            <AlertTriangle size={20} /> {whitelistError}
          </div>
        )}

        {whitelistResult && (
          <div className="mt-6 p-4 bg-[#0b0b10] border border-[#222235] rounded-md space-y-2 text-sm">
            <h4 className="text-lg font-semibold text-red-400">Whitelisted Users:</h4>
            {whitelistResult.length === 0 ? (
              <p className="text-gray-500">No users found in the whitelist.</p>
            ) : (
              <ul className="list-disc list-inside space-y-1 text-gray-300">
                {whitelistResult.map((user, index) => (
                  <li key={index} className="font-mono break-all">{user}</li>
                ))}
              </ul>
            )}
            <p className="text-gray-500 mt-2">
              <Shield size={16} className="inline-block mr-1" />
              Privacy Note: Only User IDs are listed. No other personal information is exposed.
            </p>
          </div>
        )}
      </div>
    </div>
  );
};

export default KYC;
