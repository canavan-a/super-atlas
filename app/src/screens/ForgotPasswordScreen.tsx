import { useNavigation } from '@react-navigation/native';
import { useState } from 'react';
import { ActivityIndicator, Pressable, Text, TextInput, View } from 'react-native';
import { apiJson } from '../api/client';
import { colors } from '../theme/colors';

export default function ForgotPasswordScreen() {
  const navigation = useNavigation<any>();
  const [step, setStep] = useState<'email' | 'code'>('email');
  const [email, setEmail] = useState('');
  const [code, setCode] = useState('');
  const [newPassword, setNewPassword] = useState('');
  const [error, setError] = useState<string | null>(null);
  const [notice, setNotice] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);

  const sendCode = async () => {
    setError(null);
    setSubmitting(true);
    try {
      await apiJson('/auth/forgot-password', {
        method: 'POST',
        body: JSON.stringify({ email: email.trim().toLowerCase() }),
      });
      setStep('code');
      setNotice('If an account exists for that email, a reset code is on its way.');
    } catch (e: any) {
      setError(e.message ?? 'Could not send reset code');
    } finally {
      setSubmitting(false);
    }
  };

  const resetPassword = async () => {
    setError(null);
    setSubmitting(true);
    try {
      await apiJson('/auth/reset-password', {
        method: 'POST',
        body: JSON.stringify({ email: email.trim().toLowerCase(), code: code.trim(), newPassword }),
      });
      navigation.navigate('Login');
    } catch (e: any) {
      setError(e.message ?? 'Could not reset password');
    } finally {
      setSubmitting(false);
    }
  };

  const onEmailStep = step === 'email';
  const disabled = submitting || (onEmailStep ? !email : !code || newPassword.length < 8);

  return (
    <View style={{ flex: 1, backgroundColor: colors.background, padding: 24, justifyContent: 'center' }}>
      <Text style={{ color: colors.text, fontSize: 28, fontWeight: '600', marginBottom: 32 }}>Reset password</Text>

      {onEmailStep ? (
        <TextInput
          placeholder="Email"
          placeholderTextColor={colors.textMuted}
          autoCapitalize="none"
          keyboardType="email-address"
          value={email}
          onChangeText={setEmail}
          style={inputStyle}
        />
      ) : (
        <>
          {notice && <Text style={{ color: colors.textMuted, marginBottom: 16 }}>{notice}</Text>}
          <TextInput
            placeholder="6-digit code"
            placeholderTextColor={colors.textMuted}
            keyboardType="number-pad"
            maxLength={6}
            value={code}
            onChangeText={setCode}
            style={inputStyle}
          />
          <TextInput
            placeholder="New password (8+ characters)"
            placeholderTextColor={colors.textMuted}
            secureTextEntry
            value={newPassword}
            onChangeText={setNewPassword}
            style={[inputStyle, { marginTop: 12 }]}
          />
        </>
      )}

      {error && <Text style={{ color: colors.danger, marginTop: 12 }}>{error}</Text>}

      <Pressable
        onPress={onEmailStep ? sendCode : resetPassword}
        disabled={disabled}
        style={{
          backgroundColor: colors.accent,
          borderRadius: 12,
          paddingVertical: 14,
          marginTop: 24,
          alignItems: 'center',
          opacity: disabled ? 0.5 : 1,
        }}
      >
        {submitting ? (
          <ActivityIndicator color="#fff" />
        ) : (
          <Text style={{ color: '#fff', fontWeight: '600' }}>{onEmailStep ? 'Send reset code' : 'Set new password'}</Text>
        )}
      </Pressable>

      {!onEmailStep && (
        <Pressable onPress={sendCode} disabled={submitting} style={{ marginTop: 20, alignItems: 'center' }}>
          <Text style={{ color: colors.textMuted }}>
            Didn't get it? <Text style={{ color: colors.accent }}>Resend code</Text>
          </Text>
        </Pressable>
      )}

      <Pressable onPress={() => navigation.navigate('Login')} style={{ marginTop: 20, alignItems: 'center' }}>
        <Text style={{ color: colors.textMuted }}>
          Remembered it? <Text style={{ color: colors.accent }}>Sign in</Text>
        </Text>
      </Pressable>
    </View>
  );
}

const inputStyle = {
  backgroundColor: colors.surface,
  borderColor: colors.border,
  borderWidth: 1,
  borderRadius: 12,
  paddingHorizontal: 16,
  paddingVertical: 14,
  color: colors.text,
  fontSize: 16,
};
