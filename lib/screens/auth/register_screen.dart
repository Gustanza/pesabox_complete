import 'package:flutter/material.dart';
import 'package:google_fonts/google_fonts.dart';

import '../../services/api_client.dart';
import '../../theme/app_theme.dart';
import '../../router/app_router.dart';
import 'auth_widgets.dart';

class RegisterScreen extends StatefulWidget {
  const RegisterScreen({super.key});

  @override
  State<RegisterScreen> createState() => _RegisterScreenState();
}

class _RegisterScreenState extends State<RegisterScreen> {
  final _nameController = TextEditingController();
  final _phoneController = TextEditingController();
  final _passwordController = TextEditingController();
  bool _loading = false;

  @override
  void dispose() {
    _nameController.dispose();
    _phoneController.dispose();
    _passwordController.dispose();
    super.dispose();
  }

  Future<void> _register() async {
    final name = _nameController.text.trim();
    final phone = _phoneController.text.trim();
    if (name.isEmpty || phone.isEmpty) {
      _toast('Enter your full name and phone number.');
      return;
    }
    setState(() => _loading = true);
    try {
      await api.post('/auth/register', {
        'name': name,
        'phone': phone,
        'groupName': 'My Vikoba Group',
        'password': _passwordController.text,
      });
      if (!mounted) return;
      Navigator.of(context).pushNamed(AppRouter.otp);
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
            padding: const EdgeInsets.fromLTRB(18, 6, 18, 32),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.stretch,
              children: [
                const AuthHeader(
                  dark: true,
                  title: 'Create your account',
                  subtitle: 'Join PesaBox and start managing your group easily.',
                ),
                const SizedBox(height: 28),
                AuthTextField(
                  label: 'Full name',
                  hint: 'Enter your full name',
                  controller: _nameController,
                ),
                const SizedBox(height: 14),
                AuthTextField(
                  label: 'Phone number',
                  hint: '+255 7xx xxx xxx',
                  controller: _phoneController,
                  keyboardType: TextInputType.phone,
                ),
                const SizedBox(height: 14),
                AuthTextField(
                  label: 'Password',
                  hint: 'Create a password',
                  obscure: true,
                  controller: _passwordController,
                ),
                const SizedBox(height: 30),
                PrimaryButton(
                  text: _loading ? 'Registering…' : 'Register',
                  onPressed: _loading ? null : _register,
                ),
                const SizedBox(height: 16),
                Center(
                  child: GestureDetector(
                    onTap: () =>
                        Navigator.pushNamed(context, AppRouter.login),
                    child: Padding(
                      padding: const EdgeInsets.symmetric(
                        horizontal: 8,
                        vertical: 6,
                      ),
                      child: Text.rich(
                        TextSpan(
                          text: 'Already have an account? ',
                          style: GoogleFonts.inter(
                            fontSize: 12.5,
                            color: const Color(0xFFBFE0D3),
                          ),
                          children: [
                            TextSpan(
                              text: 'Login',
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