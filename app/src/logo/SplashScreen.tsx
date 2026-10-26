import React, {useEffect, useMemo, useRef, useState} from 'react';
import {AccessibilityInfo, Animated, Easing, Platform, Pressable, StyleSheet, View, useWindowDimensions} from 'react-native';

import type {Theme} from '../theme';
import {LOGO_H, LOGO_W, PAPER_STRIPS, PIXEL_W, ToneLayer} from './LogoArt';
import {LOGO, ToneRuns} from './logoData';
import {toneColor} from './logoTheme';

// The launch screen: a scroll (the launcher icon's) unrolls from the middle
// to show the stacked "Super Atlas" wordmark, rolls shut, then unrolls again
// to show a map of the world, recolored for the current theme.
//
// Everything animated is an opacity or a translate on the native driver, and
// nothing else: no clipping, no layers resized. The rollers slide apart; the
// paper between them is cut into thin strips that each fade in as a roller
// uncovers it (their edges are always under a roller, so the fade never
// shows).

/** Timeline, in milliseconds. */
export const SPLASH_TIMING = {
  start: 250, // closed, before the first unroll
  open: 700,
  holdText: 900,
  close: 550,
  gap: 120, // closed, between the two
  holdMap: 800,
  exit: 350,
};

/** Total running time (the scroll unrolls twice). */
export const SPLASH_TOTAL = Object.values(SPLASH_TIMING).reduce((a, b) => a + b, 0) + SPLASH_TIMING.open;

interface Props {
  theme: Theme;
  /** called when the splash has finished (or was tapped away) */
  onDone: () => void;
  /** skip the motion: show the open scroll briefly, then fade */
  reduceMotion?: boolean;
  /** the app is still loading: stay up past the timeline until this goes false */
  hold?: boolean;
}

export function SplashScreen({theme, onDone, reduceMotion, hold = false}: Props): React.JSX.Element {
  const {width: screenW, height: screenH} = useWindowDimensions();
  // Kept modest, with plenty of space around it: at most 70% of the width
  // and half the height.
  const W = Math.min(screenW * 0.7, 340, (screenH * 0.5 * LOGO_W) / LOGO_H);
  const scale = W / LOGO_W;
  const H = LOGO_H * scale;
  const useNative = Platform.OS !== 'web';

  const [reduce, setReduce] = useState(reduceMotion ?? false);
  useEffect(() => {
    if (reduceMotion !== undefined) return;
    let alive = true;
    // Promise.resolve: never trust a platform API to hand back a promise.
    Promise.resolve(AccessibilityInfo.isReduceMotionEnabled?.())
      .then(v => alive && setReduce(!!v))
      .catch(() => {});
    return () => {
      alive = false;
    };
  }, [reduceMotion]);

  // Created once. (useRef({...}) would build a fresh set of Animated.Values on
  // every render and throw all but the first away.) text and map are how far
  // the scroll is open on each; only one is ever above 0, so their sum is how
  // far the rollers are apart.
  const values = useRef<{text: Animated.Value; map: Animated.Value; exit: Animated.Value} | null>(null);
  if (!values.current) {
    values.current = {text: new Animated.Value(reduce ? 1 : 0), map: new Animated.Value(0), exit: new Animated.Value(1)};
  }
  const v = values.current;

  const finished = useRef(false);
  const skipping = useRef(false);
  const running = useRef<Animated.CompositeAnimation | null>(null);
  const done = () => {
    if (finished.current) return;
    finished.current = true;
    onDone();
  };
  // A tap fades the splash out rather than unmounting it on the spot: pulling
  // views out from under natively-driven animations that are still running is
  // a classic way to crash React Native on Android.
  const skip = () => {
    if (finished.current || skipping.current) return;
    skipping.current = true;
    running.current?.stop();
    Animated.timing(v.exit, {toValue: 0, duration: 160, easing: Easing.linear, useNativeDriver: useNative}).start(done);
  };

  const timing = (val: Animated.Value, to: number, duration: number, easing = Easing.inOut(Easing.cubic)) =>
    Animated.timing(val, {toValue: to, duration, easing, useNativeDriver: useNative});

  // After the intro the splash fades out, unless the app is still loading
  // behind it (`hold`): then the map stays up until it lets go. Refs, not
  // state, so the fade starts in the same tick the intro ends.
  const holdRef = useRef(hold);
  holdRef.current = hold;
  const introDone = useRef(false);
  const exiting = useRef(false);
  const exitNow = () => {
    if (exiting.current || skipping.current) return;
    exiting.current = true;
    running.current?.stop();
    const exit = timing(v.exit, 0, reduce ? 200 : SPLASH_TIMING.exit, Easing.in(Easing.quad));
    running.current = exit;
    exit.start(r => r.finished && done());
  };
  const afterIntro = () => {
    introDone.current = true;
    if (!holdRef.current) exitNow();
  };

  // The intro: unroll to the wordmark, roll up, unroll to the map.
  useEffect(() => {
    const T = SPLASH_TIMING;
    const animation = reduce
      ? Animated.delay(800)
      : Animated.sequence([
          Animated.delay(T.start),
          timing(v.text, 1, T.open),
          Animated.delay(T.holdText),
          timing(v.text, 0, T.close),
          Animated.delay(T.gap),
          timing(v.map, 1, T.open),
          Animated.delay(T.holdMap),
        ]);
    running.current = animation;
    animation.start(r => r.finished && afterIntro());
    return () => {
      running.current?.stop();
      running.current = null;
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [reduce]);

  // Loading finished while the splash was holding: fade out now.
  useEffect(() => {
    if (!hold && introDone.current) exitNow();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [hold]);

  const fill = useMemo(() => (t: ToneRuns) => toneColor(theme, t.u, t.w), [theme]);

  // Built once so their native nodes aren't recreated.
  const anim = useMemo(() => {
    const travel = LOGO.travel * PIXEL_W * scale;
    const open = Animated.add(v.text, v.map);
    const reveal = (val: Animated.Value) =>
      PAPER_STRIPS.map(s => val.interpolate({inputRange: [s.from, s.to], outputRange: [0, 1], extrapolate: 'clamp'}));
    return {
      left: open.interpolate({inputRange: [0, 1], outputRange: [travel, 0]}),
      right: open.interpolate({inputRange: [0, 1], outputRange: [-travel, 0]}),
      text: reveal(v.text),
      map: reveal(v.map),
    };
  }, [v, scale]);

  const strips = (which: 'text' | 'map') =>
    PAPER_STRIPS.map((s, i) => {
      const tones = s[which];
      if (!tones.length) return null;
      const left = s.x0 * PIXEL_W * scale;
      const width = (s.x1 - s.x0) * PIXEL_W * scale;
      return (
        <Animated.View key={`${which}${i}`} pointerEvents="none" style={[styles.abs, {left, width, height: H, opacity: anim[which][i]}]}>
          <ToneLayer tones={tones} fill={fill} view={[s.x0 * PIXEL_W, 0, (s.x1 - s.x0) * PIXEL_W, LOGO_H]} width={width} height={H} />
        </Animated.View>
      );
    });

  const roller = (tones: ToneRuns[], translateX: Animated.AnimatedInterpolation<number>) => (
    <Animated.View pointerEvents="none" style={[styles.abs, {width: W, height: H, transform: [{translateX}]}]}>
      <ToneLayer tones={tones} fill={fill} view={[0, 0, LOGO_W, LOGO_H]} width={W} height={H} />
    </Animated.View>
  );

  return (
    <Animated.View style={[StyleSheet.absoluteFill, {backgroundColor: theme.bg, opacity: v.exit}]} accessible accessibilityLabel="Super Atlas">
      <Pressable style={styles.center} onPress={skip} accessibilityRole="button" accessibilityLabel="Skip intro">
        <View style={{width: W, height: H}}>
          {strips('text')}
          {strips('map')}
          {roller(LOGO.rollerLeft, anim.left)}
          {roller(LOGO.rollerRight, anim.right)}
        </View>
      </Pressable>
    </Animated.View>
  );
}

const styles = StyleSheet.create({
  center: {flex: 1, alignItems: 'center', justifyContent: 'center'},
  abs: {position: 'absolute', top: 0, left: 0},
});
