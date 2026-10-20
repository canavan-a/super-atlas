import {
  Check,
  ChevronLeft,
  Focus,
  Ghost,
  LucideIcon,
  Mail,
  RefreshCw,
  Settings,
  Share2,
  Users,
  X,
} from 'lucide-react-native';

// lucide-react-native is the icon set used everywhere, as in super-badger and
// super-chat: one standard library instead of hand-drawn or emoji glyphs.
export type IconName =
  | 'friends'
  | 'gear'
  | 'fit'
  | 'back'
  | 'close'
  | 'share'
  | 'check'
  | 'ghost'
  | 'mail'
  | 'refresh';

const ICONS: Record<IconName, LucideIcon> = {
  friends: Users,
  gear: Settings,
  fit: Focus,
  back: ChevronLeft,
  close: X,
  share: Share2,
  check: Check,
  ghost: Ghost,
  mail: Mail,
  refresh: RefreshCw,
};

export function Icon({
  name,
  size = 20,
  color = '#000',
  strokeWidth = 2,
}: {
  name: IconName;
  size?: number;
  color?: string;
  strokeWidth?: number;
}) {
  const IconComponent = ICONS[name];
  return <IconComponent size={size} color={color} strokeWidth={strokeWidth} />;
}
