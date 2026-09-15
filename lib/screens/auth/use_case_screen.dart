import 'package:flutter/material.dart';
import 'package:google_fonts/google_fonts.dart';

import '../../theme/app_theme.dart';
import '../../router/app_router.dart';
import 'auth_widgets.dart';

class UseCaseScreen extends StatelessWidget {
  const UseCaseScreen({super.key});

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      backgroundColor: AppColors.cream,
      body: SafeArea(
        child: SingleChildScrollView(
          padding: const EdgeInsets.fromLTRB(18, 6, 18, 32),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              const AuthHeader(
                title: 'How will you use PesaBox?',
                subtitle: 'Select the option that best describes you.',
              ),
              const SizedBox(height: 6),
              Material(
                color: Colors.transparent,
                child: InkWell(
                  onTap: () =>
                      Navigator.pushNamed(context, AppRouter.awaitingAssignment),
                  borderRadius: BorderRadius.circular(22),
                  child: Container(
                    padding: const EdgeInsets.all(16),
                    decoration: BoxDecoration(
                      color: AppColors.green100,
                      borderRadius: BorderRadius.circular(22),
                      border: Border.all(
                        color: AppColors.green600,
                        width: 2,
                      ),
                      boxShadow: const [
                        BoxShadow(
                          color: Color(0x1A0B4A3D),
                          blurRadius: 28,
                          offset: Offset(0, 12),
                        ),
                      ],
                    ),
                    child: Row(
                      children: [
                        const _UseCaseIcon(
                          background: AppColors.green600,
                          foreground: AppColors.white,
                          icon: Icons.group_outlined,
                        ),
                        const SizedBox(width: 12),
                        Expanded(
                          child: Column(
                            crossAxisAlignment: CrossAxisAlignment.start,
                            children: [
                              Text(
                                'Savings Groups',
                                style: GoogleFonts.inter(
                                  fontSize: 14,
                                  fontWeight: FontWeight.w800,
                                  color: AppColors.ink900,
                                ),
                              ),
                              const SizedBox(height: 2),
                              Text(
                                'Manage your group, meetings and finances.',
                                style: GoogleFonts.inter(
                                  fontSize: 12,
                                  color: AppColors.ink400,
                                ),
                              ),
                            ],
                          ),
                        ),
                        const Icon(
                          Icons.check,
                          color: AppColors.green600,
                          size: 18,
                        ),
                      ],
                    ),
                  ),
                ),
              ),
              const SizedBox(height: 14),
              const Opacity(
                opacity: 0.55,
                child: _UseCaseCard(
                  title: 'Other',
                  subtitle: 'Not available in MVP.',
                ),
              ),
              const SizedBox(height: 32),
              PrimaryButton(
                text: 'Continue',
                onPressed: () =>
                    Navigator.pushNamed(context, AppRouter.awaitingAssignment),
              ),
            ],
          ),
        ),
      ),
    );
  }
}

class _UseCaseIcon extends StatelessWidget {
  final Color background;
  final Color foreground;
  final IconData icon;

  const _UseCaseIcon({
    required this.background,
    required this.foreground,
    required this.icon,
  });

  @override
  Widget build(BuildContext context) {
    return Container(
      width: 42,
      height: 42,
      decoration: BoxDecoration(
        color: background,
        shape: BoxShape.circle,
      ),
      child: Icon(icon, color: foreground, size: 20),
    );
  }
}

class _UseCaseCard extends StatelessWidget {
  final String title;
  final String subtitle;

  const _UseCaseCard({required this.title, required this.subtitle});

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.all(16),
      decoration: BoxDecoration(
        color: AppColors.white,
        borderRadius: BorderRadius.circular(22),
        border: Border.all(color: AppColors.line, width: 1),
      ),
      child: Row(
        children: [
          const _UseCaseIcon(
            background: Color(0xFFEEF1F0),
            foreground: AppColors.ink400,
            icon: Icons.person_outline,
          ),
          const SizedBox(width: 12),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(
                  title,
                  style: GoogleFonts.inter(
                    fontSize: 14,
                    fontWeight: FontWeight.w700,
                    color: AppColors.ink900,
                  ),
                ),
                const SizedBox(height: 2),
                Text(
                  subtitle,
                  style: GoogleFonts.inter(
                    fontSize: 12,
                    color: AppColors.ink400,
                  ),
                ),
              ],
            ),
          ),
        ],
      ),
    );
  }
}