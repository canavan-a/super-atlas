import { useNavigation } from '@react-navigation/native';
import { useState } from 'react';
import { ActivityIndicator, Pressable, Text, TextInput, View } from 'react-native';
import { useAuth } from '../auth/AuthContext';
import { colors } from '../theme/colors';

export default function LoginScreen() {
  const { login } = useAuth();
  const navigation = useNavigation<any>();
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);

  const onSubmit = async () => {
    setError(null);
    setSubmitting(true);
    try {
      await login(email.trim().toLowerCase(), password);
    } catch (e: any) {
      setError(e.message ?? 'Login failed');
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <View style={{ flex: 1, backgroundColor: colors.background, padding: 24, justifyContent: 'center' }}>
      <Text style={{ color: colors.text, fontSize: 28, fontWeight: '600', marginBottom: 32 }}>Sign in</Text>

      <TextInput
        placeholder="Email"
        placeholderTextColor={colors.textMuted}
        autoCapitalize="none"
        keyboardType="email-address"
        value={email}
        onChangeText={setEmail}
        style={inputStyle}
      />
      <TextInput
        placeholder="Password"
        placeholderTextColor={colors.textMuted}
        secureTextEntry
        value={password}
        onChangeText={setPassword}
        style={[inputStyle, { marginTop: 12 }]}
      />

      {error && <Text style={{ color: colors.danger, marginTop: 12 }}>{error}</Text>}

      <Pressable
        onPress={onSubmit}
        disabled={submitting || !email || !password}
        style={{
          backgroundColor: colors.accent,
          borderRadius: 12,
          paddingVertical: 14,
          marginTop: 24,
          alignItems: 'center',
          opacity: submitting || !email || !password ? 0.5 : 1,
        }}
      >
        {submitting ? <ActivityIndicator color="#fff" /> : <Text style={{ color: '#fff', fontWeight: '600' }}>Sign in</Text>}
      </Pressable>

      <Pressable onPress={() => navigation.navigate('Signup')} style={{ marginTop: 20, alignItems: 'center' }}>
        <Text style={{ color: colors.textMuted }}>
          No account? <Text style={{ color: colors.accent }}>Create one</Text>
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
