import { useNavigation } from '@react-navigation/native';
import { useState } from 'react';
import { ActivityIndicator, Pressable, Text, TextInput, View } from 'react-native';
import { useAuth } from '../auth/AuthContext';
import { colors } from '../theme/colors';

export default function SignupScreen() {
  const { signup } = useAuth();
  const navigation = useNavigation<any>();
  const [displayName, setDisplayName] = useState('');
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);

  const onSubmit = async () => {
    setError(null);
    setSubmitting(true);
    try {
      await signup(email.trim().toLowerCase(), password, displayName.trim());
    } catch (e: any) {
      setError(e.message ?? 'Signup failed');
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <View style={{ flex: 1, backgroundColor: colors.background, padding: 24, justifyContent: 'center' }}>
      <Text style={{ color: colors.text, fontSize: 28, fontWeight: '600', marginBottom: 32 }}>Create account</Text>

      <TextInput
        placeholder="Name"
        placeholderTextColor={colors.textMuted}
        value={displayName}
        onChangeText={setDisplayName}
        style={inputStyle}
      />
      <TextInput
        placeholder="Email"
        placeholderTextColor={colors.textMuted}
        autoCapitalize="none"
        keyboardType="email-address"
        value={email}
        onChangeText={setEmail}
        style={[inputStyle, { marginTop: 12 }]}
      />
      <TextInput
        placeholder="Password (8+ characters)"
        placeholderTextColor={colors.textMuted}
        secureTextEntry
        value={password}
        onChangeText={setPassword}
        style={[inputStyle, { marginTop: 12 }]}
      />

      {error && <Text style={{ color: colors.danger, marginTop: 12 }}>{error}</Text>}

      <Pressable
        onPress={onSubmit}
        disabled={submitting || !email || password.length < 8 || !displayName}
        style={{
          backgroundColor: colors.accent,
          borderRadius: 12,
          paddingVertical: 14,
          marginTop: 24,
          alignItems: 'center',
          opacity: submitting || !email || password.length < 8 || !displayName ? 0.5 : 1,
        }}
      >
        {submitting ? <ActivityIndicator color="#fff" /> : <Text style={{ color: '#fff', fontWeight: '600' }}>Create account</Text>}
      </Pressable>

      <Pressable onPress={() => navigation.navigate('Login')} style={{ marginTop: 20, alignItems: 'center' }}>
        <Text style={{ color: colors.textMuted }}>
          Already have an account? <Text style={{ color: colors.accent }}>Sign in</Text>
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
