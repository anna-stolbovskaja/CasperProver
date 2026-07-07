import React from 'react';
import { Info } from 'lucide-react';

interface SectionIntroProps {
  title: string;
  description: string;
  dataSource: string;
  /** e.g. "Live API data" or "Real testnet contracts" */
  badge?: string;
  badgeColor?: 'green' | 'blue' | 'yellow';
}

const colors = {
  green: 'bg-green-900/30 text-green-400 border-green-700/40',
  blue: 'bg-blue-900/30 text-blue-400 border-blue-700/40',
  yellow: 'bg-yellow-900/30 text-yellow-400 border-yellow-700/40',
};

const SectionIntro: React.FC<SectionIntroProps> = ({
  title,
  description,
  dataSource,
  badge = 'Live data',
  badgeColor = 'green',
}) => (
  <div className="mb-6 p-4 bg-[#1a1a2a] rounded-lg border border-[#222235]">
    <div className="flex items-start gap-3">
      <Info className="w-5 h-5 text-gray-400 mt-0.5 shrink-0" />
      <div className="space-y-1.5">
        <div className="flex items-center gap-2 flex-wrap">
          <h3 className="text-sm font-semibold text-gray-200">{title}</h3>
          <span className={`text-[10px] px-1.5 py-0.5 rounded-full border font-medium ${colors[badgeColor]}`}>
            {badge}
          </span>
        </div>
        <p className="text-xs text-gray-400 leading-relaxed">{description}</p>
        <p className="text-[11px] text-gray-500">
          <span className="text-gray-400 font-medium">Data source:</span> {dataSource}
        </p>
      </div>
    </div>
  </div>
);

export default SectionIntro;
