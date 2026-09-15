import 'package:flutter/material.dart';
import 'package:google_fonts/google_fonts.dart';

import '../../services/api_client.dart';
import '../../services/app_data.dart';
import '../../theme/app_theme.dart';
import '../../router/app_router.dart';
import 'auth_widgets.dart';

class LoginScreen extends StatefulWidget {
  const LoginScreen({super.key});

  @override
  State<LoginScreen> createState() => _LoginScreenState();
}

class _LoginScreenState extends State<LoginScreen> {
  final _phoneController = TextEditingController();
  final _passwordController = TextEditingController();
  bool _loading = false;

  @override
  void dispose() {
    _phoneController.dispose();
    _passwordController.dispose();
    super.dispose();
  }

  Future<void> _login() async {
    final phone = _phoneController.text.trim();
    final password = _passwordController.text;
    if (phone.isEmpty || password.isEmpty) {
      _toast('Enter your phone number and password.');
      return;
    }
    setState(() => _loading = true);
    try {
      final res = await api.post('/auth/login', {
        'phone': phone,
        'password': password,
      });
      final token = res['token'] as String?;
      final user = (res['user'] as Map?)?.cast<String, dynamic>();
      AppState.I.token = token;
      AppState.I.user = user;
      if (!mounted) return;
      Navigator.of(context)
          .pushNamedAndRemoveUntil(AppRouter.dashboard, (r) => false);
    } on ApiException catch (e) {
      if (!mounted) return;
      setState(() => _loading = false);
      _toast(e.message);
    } catch (_) {
      if (!mounted) return;
      setState(() => _loading = false);
      _toast('Could not reach the server. Check your connection.');
    }
  }

  void _toast(String message) {
    ScaffoldMessenger.of(context).showSnackBar(
      SnackBar(content: Text(message)),
    );
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      body: Container(
        decoration: const BoxDecoration(
          gradient: LinearGradient(
            begin: Alignment.topCenter,
            end: Alignment.bottomCenter,
            colors: [AppColors.teal900, AppColors.teal800],
          ),
        ),
        child: SafeArea(
          child: SingleChildScrollView(
            padding: const EdgeInsets.fromLTRB(18, 10, 18, 32),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.stretch,
              children: [
                const BrandMark(size: 60, radius: 16, fontSize: 26),
                const SizedBox(height: 18),
                Text(
                  'Login to your account',
                  style: GoogleFonts.plusJakartaSans(
                    fontSize: 19,
                    fontWeight: FontWeight.w800,
                    color: AppColors.white,
                  ),
                ),
                const SizedBox(height: 20),
                AuthTextField(
                  label: 'Phone number',
                  hint: '+255 7xx xxx xxx',
                  controller: _phoneController,
                  keyboardType: TextInputType.phone,
                ),
                const SizedBox(height: 14),
                AuthTextField(
                  label: 'Password',
                  hint: 'Enter your password',
                  obscure: true,
                  controller: _passwordController,
                ),
                const SizedBox(height: 6),
                Align(
                  alignment: Alignment.centerRight,
                  child: GestureDetector(
                    onTap: () =>
                        Navigator.pushNamed(context, AppRouter.forgotPassword),
                    child: Padding(
                      padding: const EdgeInsets.symmetric(
                        horizontal: 4,
                        vertical: 6,
                      ),
                      child: Text(
                        'Forgot password?',
                        style: GoogleFonts.inter(
                          fontSize: 12,
                          color: const Color(0xFFBFE0D3),
                        ),
                      ),
                    ),
                  ),
                ),
                const SizedBox(height: 18),
                PrimaryButton(
                  text: _loading ? 'Logging in…' : 'Login',
                  onPressed: _loading ? null : _login,
                ),
                const SizedBox(height: 10),
                OutlineButton(
                  text: 'Login with OTP',
                  onPressed: () => Navigator.pushNamed(context, AppRouter.otp),
                ),
                const SizedBox(height: 16),
                Center(
                  child: GestureDetector(
                    onTap: () =>
                        Navigator.pushNamed(context, AppRouter.register),
                    child: Padding(
                      padding: const EdgeInsets.symmetric(
                        horizontal: 8,
                        vertical: 6,
                      ),
                      child: Text.rich(
                        TextSpan(
                          text: "Don't have an account? ",
                          style: GoogleFonts.inter(
                            fontSize: 12.5,
                            color: const Color(0xFFBFE0D3),
                          ),
                          children: [
                            TextSpan(
                              text: 'Register',
                              style: GoogleFonts.inter(
                                fontSize: 12.5,
                                fontWeight: FontWeight.w700,
                                color: AppColors.white,
                              ),
                            ),
                          ],
                        ),
                      ),
                    ),
                  ),
                ),
              ],
            ),
          ),
        ),
      ),
    );
  }
}