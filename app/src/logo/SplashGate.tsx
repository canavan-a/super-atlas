import React, {useEffect, useState} from 'react';
import {StyleSheet, View} from 'react-native';

import {settingsStore} from '../settings';
import {useTheme} from '../theme';
import {SplashScreen} from './SplashScreen';
import {splashHold} from './splashHold';

// Plays the launch splash over the app, once per process. The app renders
// underneath from the start (so it's ready the moment the splash lifts); the
// splash waits for the saved settings first, so it opens in the theme you
// picked instead of flashing the default one.

// Module-level so a remount (e.g. a fast refresh, or the tree being rebuilt
// after a config change) doesn't replay it; a fresh launch is a fresh process.
let played = false;
const playedListeners = new Set<() => void>();

function markPlayed(): void {
  played = true;
  playedListeners.forEach(l => l());
}

/** For tests: forget that the splash has played. */
export function resetSplashForTests(): void {
  played = false;
}

/**
 * True once the splash is out of the way (finished, skipped or failed). The
 * app renders under the splash, so screens use this to hold off on things
 * like focusing a text field, which would pop the keyboard up over the logo.
 */
export function useSplashDone(): boolean {
  const [done, setDone] = useState(played);
  useEffect(() => {
    if (played) {
      setDone(true);
      return;
    }
    const l = () => setDone(true);
    playedListeners.add(l);
    return () => {
      playedListeners.delete(l);
    };
  }, []);
  return done;
}

type Phase = 'wait' | 'play' | 'done';

/** Consecutive launches whose splash didn't finish before it turns itself off. */
export const MAX_SPLASH_STRIKES = 2;

/**
 * The splash is decoration: if anything in it throws, drop it and let the app
 * through rather than taking the whole app down with it.
 */
export class SplashBoundary extends React.Component<{onFail: () => void; children: React.ReactNode}, {failed: boolean}> {
  state = {failed: false};
  static getDerivedStateFromError(): {failed: boolean} {
    return {failed: true};
  }
  componentDidCatch(): void {
    this.props.onFail();
  }
  render(): React.ReactNode {
    return this.state.failed ? null : this.props.children;
  }
}

export function SplashGate({children}: {children: React.ReactNode}): React.JSX.Element {
  const theme = useTheme();
  const [phase, setPhase] = useState<Phase>(played ? 'done' : 'wait');
  const [hold, setHold] = useState(splashHold.get());
  useEffect(() => splashHold.subscribe(setHold), []);

  useEffect(() => {
    if (phase !== 'wait') return;
    let alive = true;
    const skipToApp = () => {
      markPlayed();
      setPhase('done');
    };
    settingsStore
      .load()
      .then(async settings => {
        if (!alive) return;
        if (settings.launchSplash === false) return skipToApp(); // turned off in Settings
        const strikes = settings.splashStrikes ?? 0;
        if (strikes >= MAX_SPLASH_STRIKES) {
          // The last launches never got to the end of the animation (a crash,
          // most likely). Don't crash a third time: turn it off and say so.
          await settingsStore.save({...settings, launchSplash: false, splashStrikes: 0, splashAutoDisabled: true});
          if (alive) skipToApp();
          return;
        }
        // Recorded *before* the animation starts, so a crash mid-animation
        // still leaves the strike behind.
        await settingsStore.save({...settings, splashStrikes: strikes + 1});
        if (alive) setPhase('play');
      })
      .catch(() => alive && skipToApp()); // never let storage trouble block the app
    return () => {
      alive = false;
    };
  }, [phase]);

  return (
    <View style={styles.fill}>
      {children}
      {phase === 'play' && (
        <SplashBoundary
          onFail={() => {
            markPlayed();
            setPhase('done');
          }}>
          <SplashScreen
            theme={theme}
            hold={hold}
            onDone={() => {
              // It got to the end: clear the strike.
              settingsStore.save({...settingsStore.get(), splashStrikes: 0}).catch(() => {});
              markPlayed();
              setPhase('done');
            }}
          />
        </SplashBoundary>
      )}
    </View>
  );
}

const styles = StyleSheet.create({fill: {flex: 1}});
