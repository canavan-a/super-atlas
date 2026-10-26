import { useEffect, useRef, useState } from 'react';
import { KeyboardAvoidingView, Platform, Pressable, StyleSheet, Text, TextInput, View } from 'react-native';
import { useAuth } from '../auth/AuthContext';
import { Button, TextField } from '../components/ui';
import { useSplashDone } from '../logo/SplashGate';
import { Theme, useThemedStyles } from '../theme';

const RESEND_AFTER_S = 60;

// One screen for both signing in and signing up: email, then the emailed
// code, then a name only if this email has no account yet.
export default function SignInScreen() {
  const { requestCode, verify } = useAuth();
  const styles = useThemedStyles(makeStyles);
  const [step, setStep] = useState<'email' | 'code' | 'name'>('email');
  const [email, setEmail] = useState('');
  const [code, setCode] = useState('');
  const [name, setName] = useState('');
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);
  const [resendIn, setResendIn] = useState(0);
  const codeRef = useRef<TextInput>(null);
  const emailRef = useRef<TextInput>(null);
  const splashDone = useSplashDone();

  // Instead of autoFocus: this screen sits under the launch splash, and an
  // early focus would pop the keyboard up over the logo.
  useEffect(() => {
    if (splashDone && step === 'email') emailRef.current?.focus();
  }, [splashDone, step]);

  useEffect(() => {
    if (resendIn <= 0) return;
    const t = setTimeout(() => setResendIn((s) => s - 1), 1000);
    return () => clearTimeout(t);
  }, [resendIn]);

  const run = async (fn: () => Promise<void>) => {
    setError(null);
    setBusy(true);
    try {
      await fn();
    } catch (e: any) {
      setError(e.message ?? 'Something went wrong');
    } finally {
      setBusy(false);
    }
  };

  const sendCode = () =>
    run(async () => {
      await requestCode(email.trim().toLowerCase());
      setStep('code');
      setCode('');
      setResendIn(RESEND_AFTER_S);
      setTimeout(() => codeRef.current?.focus(), 100);
    });

  const submitCode = (value = code) =>
    run(async () => {
      const result = await verify(email.trim().toLowerCase(), value);
      if (result === 'needsName') setStep('name');
    });

  const submitName = () =>
    run(async () => {
      await verify(email.trim().toLowerCase(), code, name.trim());
    });

  const onCodeChange = (value: string) => {
    const digits = value.replace(/\D/g, '').slice(0, 6);
    setCode(digits);
    if (digits.length === 6) submitCode(digits);
  };

  return (
    <KeyboardAvoidingView style={styles.root} behavior={Platform.OS === 'ios' ? 'padding' : undefined}>
      <Text style={styles.title}>Super Atlas</Text>

      {step === 'email' && (
        <>
          <Text style={styles.subtitle}>See your friends on a map. Enter your email and we'll send you a code.</Text>
          <TextField
            ref={emailRef}
            placeholder="you@example.com"
            autoCapitalize="none"
            autoComplete="email"
            keyboardType="email-address"
            value={email}
            onChangeText={setEmail}
            onSubmitEditing={sendCode}
            returnKeyType="send"
          />
          <Button label="Send code" onPress={sendCode} loading={busy} disabled={!email.includes('@')} style={styles.gap} />
        </>
      )}

      {step === 'code' && (
        <>
          <Text style={styles.subtitle}>Enter the 6-digit code we sent to {email.trim()}.</Text>
          <TextField
            ref={codeRef}
            placeholder="123456"
            keyboardType="number-pad"
            autoComplete="one-time-code"
            textContentType="oneTimeCode"
            maxLength={6}
            value={code}
            onChangeText={onCodeChange}
            style={styles.codeInput}
          />
          <Button label="Continue" onPress={() => submitCode()} loading={busy} disabled={code.length !== 6} style={styles.gap} />
          <View style={styles.links}>
            <Pressable onPress={() => setStep('email')} hitSlop={8}>
              <Text style={styles.link}>Change email</Text>
            </Pressable>
            <Pressable onPress={sendCode} disabled={resendIn > 0 || busy} hitSlop={8}>
              <Text style={[styles.link, resendIn > 0 && styles.linkDisabled]}>
                {resendIn > 0 ? `Resend in ${resendIn}s` : 'Resend code'}
              </Text>
            </Pressable>
          </View>
        </>
      )}

      {step === 'name' && (
        <>
          <Text style={styles.subtitle}>Welcome! What should your friends see you as?</Text>
          <TextField
            placeholder="Your name"
            autoCapitalize="words"
            autoComplete="name"
            autoFocus
            value={name}
            onChangeText={setName}
            onSubmitEditing={submitName}
            maxLength={40}
          />
          <Button label="Get started" onPress={submitName} loading={busy} disabled={!name.trim()} style={styles.gap} />
        </>
      )}

      {error && <Text style={styles.error}>{error}</Text>}
    </KeyboardAvoidingView>
  );
}

const makeStyles = (theme: Theme) =>
  StyleSheet.create({
    root: { flex: 1, backgroundColor: theme.bg, padding: 24, justifyContent: 'center' },
    title: { color: theme.text, fontSize: 30, fontWeight: '700', marginBottom: 8 },
    subtitle: { color: theme.textMuted, fontSize: 15, marginBottom: 24, lineHeight: 21 },
    gap: { marginTop: 16 },
    codeInput: { fontSize: 24, letterSpacing: 8, textAlign: 'center' },
    links: { flexDirection: 'row', justifyContent: 'space-between', marginTop: 20 },
    link: { color: theme.primary, fontWeight: '600' },
    linkDisabled: { color: theme.textMuted },
    error: { color: theme.danger, marginTop: 16 },
  });
