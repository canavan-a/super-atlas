import React from 'react';
import { ActivityIndicator, Pressable, StyleSheet, Text, TextInput, TextInputProps, View, ViewStyle } from 'react-native';
import { Theme, useTheme, useThemedStyles } from '../theme';

// The few building blocks every screen shares, so they all look the same.

export function Button({
  label,
  onPress,
  disabled,
  loading,
  variant = 'primary',
  style,
}: {
  label: string;
  onPress: () => void;
  disabled?: boolean;
  loading?: boolean;
  variant?: 'primary' | 'secondary' | 'danger';
  style?: ViewStyle;
}) {
  const theme = useTheme();
  const styles = useThemedStyles(makeStyles);
  const bg = variant === 'primary' ? theme.primary : variant === 'danger' ? theme.dangerBg : theme.surfaceAlt;
  const fg = variant === 'primary' ? theme.primaryText : variant === 'danger' ? theme.danger : theme.text;
  return (
    <Pressable
      onPress={onPress}
      disabled={disabled || loading}
      style={({ pressed }) => [
        styles.button,
        { backgroundColor: bg, opacity: disabled ? 0.5 : pressed ? 0.8 : 1 },
        style,
      ]}
    >
      {loading ? <ActivityIndicator color={fg} /> : <Text style={[styles.buttonText, { color: fg }]}>{label}</Text>}
    </Pressable>
  );
}

export const TextField = React.forwardRef<TextInput, TextInputProps>(function TextField(props, ref) {
  const theme = useTheme();
  const styles = useThemedStyles(makeStyles);
  return <TextInput ref={ref} placeholderTextColor={theme.textMuted} {...props} style={[styles.input, props.style]} />;
});

export function Card({ children, style }: { children: React.ReactNode; style?: ViewStyle }) {
  const styles = useThemedStyles(makeStyles);
  return <View style={[styles.card, style]}>{children}</View>;
}

const makeStyles = (theme: Theme) =>
  StyleSheet.create({
    button: {
      borderRadius: 12,
      paddingVertical: 14,
      paddingHorizontal: 18,
      alignItems: 'center',
      justifyContent: 'center',
    },
    buttonText: { fontSize: 16, fontWeight: '600' },
    input: {
      backgroundColor: theme.surface,
      borderColor: theme.border,
      borderWidth: 1,
      borderRadius: 12,
      paddingHorizontal: 16,
      paddingVertical: 14,
      color: theme.text,
      fontSize: 16,
    },
    card: {
      backgroundColor: theme.surface,
      borderColor: theme.border,
      borderWidth: 1,
      borderRadius: 14,
      padding: 16,
    },
  });
