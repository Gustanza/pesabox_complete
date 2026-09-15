import 'package:flutter/material.dart';
import 'package:google_fonts/google_fonts.dart';

import '../../router/app_router.dart';
import '../../theme/app_theme.dart';
import '../auth/auth_widgets.dart';

class AddMemberScreen extends StatefulWidget {
  const AddMemberScreen({super.key});

  @override
  State<AddMemberScreen> createState() => _AddMemberScreenState();
}

class _AddMemberScreenState extends State<AddMemberScreen> {
  int _memberCount = 1;

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      backgroundColor: AppColors.cream,
      body: SafeArea(
        child: SingleChildScrollView(
          padding: const EdgeInsets.symmetric(horizontal: 20),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              const SizedBox(height: 16),
              AuthHeader(
                title: 'Add Members',
                subtitle: 'Enter member details manually.',
                onBack: () => Navigator.of(context).maybePop(),
              ),
              const SizedBox(height: 28),
              for (var i = 0; i < _memberCount; i++) ...[
                _MemberForm(memberNumber: i + 1),
                const SizedBox(height: 20),
              ],
              GestureDetector(
                onTap: () {
                  setState(() => _memberCount++);
                },
                child: Row(
                  mainAxisSize: MainAxisSize.min,
                  children: [
                    const Icon(
                      Icons.add_circle_outline,
                      size: 18,
                      color: AppColors.green600,
                    ),
                    const SizedBox(width: 6),
                    Text(
                      'Add another member',
                      style: GoogleFonts.inter(
                        fontSize: 14,
                        fontWeight: FontWeight.w600,
                        color: AppColors.green600,
                      ),
                    ),
                  ],
                ),
              ),
              const SizedBox(height: 32),
              PrimaryButton(
                text: 'Save members',
                onPressed: () {
                  Navigator.of(context)
                      .pushNamedAndRemoveUntil(
                    AppRouter.membersList,
                    (route) => route.isFirst,
                  );
                },
              ),
              const SizedBox(height: 20),
              Container(
                width: double.infinity,
                padding: const EdgeInsets.all(14),
                decoration: BoxDecoration(
                  color: AppColors.gold100,
                  borderRadius: BorderRadius.circular(12),
                ),
                child: Row(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    const Icon(
                      Icons.info_outline,
                      size: 18,
                      color: AppColors.gold500,
                    ),
                    const SizedBox(width: 10),
                    Expanded(
                      child: Text(
                        'Members receive a confirmation SMS with login credentials once added.',
                        style: GoogleFonts.inter(
                          fontSize: 12.5,
                          color: AppColors.ink700,
                          height: 1.4,
                        ),
                      ),
                    ),
                  ],
                ),
              ),
              const SizedBox(height: 32),
            ],
          ),
        ),
      ),
    );
  }
}

class _MemberForm extends StatelessWidget {
  final int memberNumber;

  const _MemberForm({required this.memberNumber});

  @override
  Widget build(BuildContext context) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text(
          'Member $memberNumber',
          style: GoogleFonts.plusJakartaSans(
            fontSize: 15,
            fontWeight: FontWeight.w700,
            color: AppColors.ink900,
          ),
        ),
        const SizedBox(height: 12),
        AuthTextField(
          label: 'Full name',
          hint: 'Enter full name',
        ),
        const SizedBox(height: 18),
        _PhoneField(),
        const SizedBox(height: 18),
        _GenderDropdown(),
        const SizedBox(height: 18),
        AuthTextField(
          label: 'Member number',
          hint: 'e.g. 001',
          requiredField: false,
        ),
      ],
    );
  }
}

class _PhoneField extends StatelessWidget {
  @override
  Widget build(BuildContext context) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text.rich(
          TextSpan(
            text: 'Phone',
            style: GoogleFonts.inter(
              fontSize: 12.5,
              fontWeight: FontWeight.w700,
              color: AppColors.ink700,
            ),
            children: [
              TextSpan(
                text: ' *',
                style: GoogleFonts.inter(color: AppColors.danger),
              ),
            ],
          ),
        ),
        const SizedBox(height: 6),
        TextField(
          keyboardType: TextInputType.phone,
          style: GoogleFonts.inter(fontSize: 14, color: AppColors.ink900),
          decoration: InputDecoration(
            hintText: '754 123 456',
            hintStyle: GoogleFonts.inter(
              fontSize: 14,
              color: AppColors.ink400,
            ),
            prefixIcon: Padding(
              padding: const EdgeInsets.only(left: 14, right: 8),
              child: Row(
                mainAxisSize: MainAxisSize.min,
                children: [
                  Text(
                    '+255',
                    style: GoogleFonts.inter(
                      fontSize: 14,
                      fontWeight: FontWeight.w600,
                      color: AppColors.ink900,
                    ),
                  ),
                  const SizedBox(width: 6),
                  Container(
                    width: 1,
                    height: 20,
                    color: AppColors.line,
                  ),
                ],
              ),
            ),
            filled: true,
            fillColor: AppColors.white,
            contentPadding:
                const EdgeInsets.symmetric(horizontal: 14, vertical: 12),
            enabledBorder: OutlineInputBorder(
              borderRadius: BorderRadius.circular(12),
              borderSide:
                  const BorderSide(color: AppColors.line, width: 1.5),
            ),
            focusedBorder: OutlineInputBorder(
              borderRadius: BorderRadius.circular(12),
              borderSide:
                  const BorderSide(color: AppColors.green600, width: 1.5),
            ),
          ),
        ),
      ],
    );
  }
}

class _GenderDropdown extends StatelessWidget {
  @override
  Widget build(BuildContext context) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text.rich(
          TextSpan(
            text: 'Gender',
            style: GoogleFonts.inter(
              fontSize: 12.5,
              fontWeight: FontWeight.w700,
              color: AppColors.ink700,
            ),
            children: [
              TextSpan(
                text: ' *',
                style: GoogleFonts.inter(color: AppColors.danger),
              ),
            ],
          ),
        ),
        const SizedBox(height: 6),
        DropdownButtonFormField<String>(
          hint: Text(
            'Select gender',
            style: GoogleFonts.inter(
              fontSize: 14,
              color: AppColors.ink400,
            ),
          ),
          items: const [
            DropdownMenuItem(value: 'Male', child: Text('Male')),
            DropdownMenuItem(value: 'Female', child: Text('Female')),
          ],
          onChanged: (_) {},
          style: GoogleFonts.inter(fontSize: 14, color: AppColors.ink900),
          decoration: InputDecoration(
            filled: true,
            fillColor: AppColors.white,
            contentPadding:
                const EdgeInsets.symmetric(horizontal: 14, vertical: 12),
            enabledBorder: OutlineInputBorder(
              borderRadius: BorderRadius.circular(12),
              borderSide:
                  const BorderSide(color: AppColors.line, width: 1.5),
            ),
            focusedBorder: OutlineInputBorder(
              borderRadius: BorderRadius.circular(12),
              borderSide:
                  const BorderSide(color: AppColors.green600, width: 1.5),
            ),
          ),
        ),
      ],
    );
  }
}
